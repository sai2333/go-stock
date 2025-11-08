package tools

import (
    "context"
    "encoding/json"
    "fmt"
    "math"
    "regexp"
    "strconv"
    "strings"
    "time"

    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/schema"
    "github.com/go-resty/resty/v2"
    "github.com/tidwall/gjson"
    "go-stock/backend/data"
    "go-stock/backend/db"
    "go-stock/backend/util"
)

// @Author spark
// @Date 2025/11/08
// @Desc 根据自然语言筛选ETF（横盘震荡/T+0等），返回TopN及净值/估算信息

func GetChoiceETFByIndicatorsTool() tool.InvokableTool { return &ChoiceETFByIndicators{} }

type ChoiceETFByIndicators struct{}

func (t ChoiceETFByIndicators) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "ChoiceETFByIndicators",
        Desc: "根据自然语言筛选ETF，例如：'T+0做T，近1月绝对值≤2%，近3月绝对值≤6%，规模≥50亿，按估算涨跌排序，前5''",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "words": {Type: "string", Desc: "筛选条件的自然语言描述", Required: true},
        }),
    }, nil
}

// 输出行结构
type etfRow struct {
    Name       string  `md:"基金名称"`
    Code       string  `md:"基金代码"`
    Type       string  `md:"基金类型"`
    Scale      string  `md:"规模(亿元)"`
    Track      string  `md:"跟踪标的"`
    M1         *float64 `md:"近1月(%)"`
    M3         *float64 `md:"近3月(%)"`
    Dwjz       string  `md:"单位净值"`
    Jzrq       string  `md:"净值日期"`
    Gsz        string  `md:"估算净值"`
    Gszzl      string  `md:"估算涨跌(%)"`
    Gztime     string  `md:"估算时间"`
}

func (t ChoiceETFByIndicators) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
    words := gjson.Get(argumentsInJSON, "words").String()
    if strings.TrimSpace(words) == "" {
        return "", fmt.Errorf("words不能为空")
    }

    // 解析阈值
    m1AbsMax := parseFloatWithDefault(words, `近?1月[^0-9]*([0-9]+(?:\.[0-9]+)?)%`, 2)
    m3AbsMax := parseFloatWithDefault(words, `近?3月[^0-9]*([0-9]+(?:\.[0-9]+)?)%`, 6)
    scaleMin := parseFloatWithDefault(words, `规模[^0-9]*([0-9]+)`, 50)
    topN := int(parseFloatWithDefault(words, `前\s*([0-9]+)`, 5))
    sortKey := "est"
    if regexp.MustCompile("估算|盘中").FindString(words) != "" && regexp.MustCompile("排序").FindString(words) != "" {
        sortKey = "est"
    } else if regexp.MustCompile("近1月").FindString(words) != "" && regexp.MustCompile("排序").FindString(words) != "" {
        sortKey = "m1"
    } else if regexp.MustCompile("近3月").FindString(words) != "" && regexp.MustCompile("排序").FindString(words) != "" {
        sortKey = "m3"
    }

    api := data.NewFundApi()
    // 确保有基金列表（至少含代码与名称）
    api.AllFund()

    // 初筛：名称包含ETF
    var candidates []data.FundBasic
    db.Dao.Where("name like ? OR name like ?", "%ETF%", "%指数%").Limit(600).Find(&candidates)
    if len(candidates) == 0 {
        // 兜底一：尝试关键字检索
        candidates = api.GetFundList("ETF")
        // 兜底二：若仍为空，使用常见高流动性ETF代码列表
        if len(candidates) == 0 {
            hotCodes := []string{"510300", "510500", "510050", "159915", "510880", "512800", "159949", "588000", "513500", "563000"}
            for _, c := range hotCodes {
                candidates = append(candidates, data.FundBasic{Code: c, Name: c})
            }
        }
    }

    client := resty.New()
    var rows []etfRow
    // 逐个拉基础信息并过滤
    limit := minInt(len(candidates), 200)
    for i := 0; i < limit; i++ {
        code := candidates[i].Code
        // 实时拉取基础信息（包含类型/规模/近月近季）
        fb, err := api.CrawlFundBasic(code)
        // 若解析失败，则使用候选中的基础信息兜底，仍允许进入后续流程（便于估算值排序）
        if err != nil || fb == nil {
            fb = &data.FundBasic{Code: code, Name: candidates[i].Name}
        }
        // 类型过滤放宽：候选集中已包含“ETF/指数”关键字，这里不再强校验类型
        if parseScaleToFloat(fb.Scale) < scaleMin {
            continue
        }
        // 近1月/近3月横盘（绝对值）
        if fb.NetGrowth1 != nil && math.Abs(*fb.NetGrowth1) > m1AbsMax {
            continue
        }
        if fb.NetGrowth3 != nil && math.Abs(*fb.NetGrowth3) > m3AbsMax {
            continue
        }

        // 拉取单位净值与估算净值（东财/Sina）
        dwjz, jzrq, gsz, gszzl, gztime := fetchNetValues(client, code)

        rows = append(rows, etfRow{
            Name:   fb.Name,
            Code:   fb.Code,
            Type:   fb.Type,
            Scale:  fb.Scale,
            Track:  fb.TrackingTarget,
            M1:     fb.NetGrowth1,
            M3:     fb.NetGrowth3,
            Dwjz:   dwjz,
            Jzrq:   jzrq,
            Gsz:    gsz,
            Gszzl:  gszzl,
            Gztime: gztime,
        })
    }

    // 排序
    switch sortKey {
    case "m1":
        rows = sortByFloat(rows, func(r etfRow) float64 { return ptrOrZero(r.M1) }, true)
    case "m3":
        rows = sortByFloat(rows, func(r etfRow) float64 { return ptrOrZero(r.M3) }, true)
    default: // est
        rows = sortByFloat(rows, func(r etfRow) float64 { return parseFloat(r.Gszzl) }, true)
    }

    // TopN
    if len(rows) > topN {
        rows = rows[:topN]
    }

    title := fmt.Sprintf("ETF筛选(近1月≤%.2f%%,近3月≤%.2f%%,规模≥%.0f亿,排序:%s,Top:%d)", m1AbsMax, m3AbsMax, scaleMin, sortKey, topN)
    return util.MarkdownTableWithTitle(title, rows), nil
}

// helpers
func parseFloatWithDefault(s, pattern string, def float64) float64 {
    re := regexp.MustCompile(pattern)
    m := re.FindStringSubmatch(s)
    if len(m) >= 2 {
        if v, err := strconv.ParseFloat(m[1], 64); err == nil { return v }
    }
    return def
}
func parseScaleToFloat(scale string) float64 {
    if strings.TrimSpace(scale) == "" { return 0 }
    // 只取数字部分，单位按亿元
    num := regexp.MustCompile(`[^0-9\.]+`).ReplaceAllString(scale, "")
    v, _ := strconv.ParseFloat(num, 64)
    return v
}
func fetchNetValues(client *resty.Client, code string) (dwjz, jzrq, gsz, gszzl, gztime string) {
    // 估算净值：东财
    resp1, err := client.SetTimeout(15*time.Second).R().
        SetHeader("User-Agent", "Mozilla/5.0").
        SetHeader("Referer", "https://fund.eastmoney.com/").
        SetQueryParam("rt", strconv.FormatInt(time.Now().UnixMilli(), 10)).
        Get(fmt.Sprintf("https://fundgz.1234567.com.cn/js/%s.js", code))
    if err == nil && resp1.StatusCode() == 200 {
        body := string(resp1.Body())
        if strings.Contains(body, "jsonpgz") {
            body = strings.TrimPrefix(body, "jsonpgz(")
            body = strings.TrimSuffix(body, ");")
            var est data.FundNetUnitValue
            _ = json.Unmarshal([]byte(body), &est)
            gsz, gszzl, gztime = est.Gsz, est.Gszzl, est.Gztime
        }
    }
    // 单位净值：Sina
    resp2, err := client.SetTimeout(15*time.Second).R().
        SetHeader("Host", "hq.sinajs.cn").
        SetHeader("User-Agent", "Mozilla/5.0").
        SetHeader("Referer", "https://finance.sina.com.cn").
        Get(fmt.Sprintf("http://hq.sinajs.cn/rn=%d&list=f_%s", time.Now().UnixMilli(), code))
    if err == nil && resp2.StatusCode() == 200 {
        dataStr := string(data.GB18030ToUTF8(resp2.Body()))
        parts := strings.Split(dataStr, "=")
        if len(parts) >= 2 {
            vals := strings.Split(strings.Trim(parts[1], "\"\n"), ",")
            if len(vals) >= 5 {
                dwjz = vals[1]
                jzrq = vals[4]
            }
        }
    }
    return
}
func sortByFloat(rows []etfRow, val func(etfRow) float64, desc bool) []etfRow {
    if len(rows) <= 1 { return rows }
    // 简单选择排序避免引入sort包
    for i := 0; i < len(rows); i++ {
        target := i
        for j := i + 1; j < len(rows); j++ {
            if desc {
                if val(rows[j]) > val(rows[target]) { target = j }
            } else {
                if val(rows[j]) < val(rows[target]) { target = j }
            }
        }
        rows[i], rows[target] = rows[target], rows[i]
    }
    return rows
}
func ptrOrZero(p *float64) float64 { if p == nil { return 0 }; return *p }
func parseFloat(s string) float64 { v, _ := strconv.ParseFloat(strings.TrimSpace(strings.Trim(s, "%")), 64); return v }
func minInt(a, b int) int { if a < b { return a }; return b }