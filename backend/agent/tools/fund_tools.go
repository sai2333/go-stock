package tools

import (
    "context"
    "fmt"
    "encoding/json"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/schema"
    "github.com/duke-git/lancet/v2/convertor"
    "github.com/go-resty/resty/v2"
    "github.com/tidwall/gjson"
    "go-stock/backend/data"
    "go-stock/backend/util"
    "strconv"
    "strings"
    "time"
)

// @Author spark
// @Date 2025/11/08
// @Desc 基金/ETF相关工具：关键词查询、基础信息、净值数据
//-----------------------------------------------------------------------------------

// SearchFundByKeyTool
func GetSearchFundByKeyTool() tool.InvokableTool { return &SearchFundByKeyTool{} }

type SearchFundByKeyTool struct{}

func (t SearchFundByKeyTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "SearchFundByKey",
        Desc: "按关键词查询基金列表（代码/简称）",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "key": {Type: "string", Desc: "关键词（代码/名称）", Required: true},
        }),
    }, nil
}

func (t SearchFundByKeyTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
    key := gjson.Get(argumentsInJSON, "key").String()
    if strings.TrimSpace(key) == "" {
        return "", fmt.Errorf("参数key不能为空")
    }
    funds := data.NewFundApi().GetFundList(key)
    if len(funds) == 0 {
        return fmt.Sprintf("未找到与‘%s’匹配的基金", key), nil
    }
    // 仅 Code/Name 字段通常已填充，其余字段可能为空，直接用结构体生成表格
    return util.MarkdownTableWithTitle("基金列表("+key+")", funds), nil
}

// GetFundBasicTool
func GetFundBasicTool() tool.InvokableTool { return &FundBasicTool{} }

type FundBasicTool struct{}

func (t FundBasicTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "GetFundBasic",
        Desc: "查询基金基础信息（类型、成立日期、规模、公司、经理、评级、跟踪标的、近月/季/年绩效等）",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "fundCode": {Type: "string", Desc: "基金代码", Required: true},
        }),
    }, nil
}

func (t FundBasicTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
    code := gjson.Get(argumentsInJSON, "fundCode").String()
    if strings.TrimSpace(code) == "" {
        return "", fmt.Errorf("参数fundCode不能为空")
    }
    fund, err := data.NewFundApi().CrawlFundBasic(code)
    if err != nil {
        return "", err
    }
    // 使用切片包装保证标题输出
    return util.MarkdownTableWithTitle(fund.Name+" 基金基础信息", []data.FundBasic{*fund}), nil
}

// GetFundNetValuesTool
func GetFundNetValuesTool() tool.InvokableTool { return &FundNetValuesTool{} }

type FundNetValuesTool struct{}

func (t FundNetValuesTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "GetFundNetValues",
        Desc: "查询基金净值（单位净值/净值日期）与估算净值（估算涨跌幅/时间）",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "fundCode": {Type: "string", Desc: "基金代码", Required: true},
        }),
    }, nil
}

// 组合输出的数据结构，带 md 标签用于美化表头
type fundNetValuesRow struct {
    Name    string `md:"基金名称"`
    Code    string `md:"基金代码"`
    Dwjz    string `md:"单位净值"`
    Jzrq    string `md:"净值日期"`
    Gsz     string `md:"估算净值"`
    Gszzl   string `md:"估算涨跌幅(%)"`
    Gztime  string `md:"估算时间"`
}

func (t FundNetValuesTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
    code := gjson.Get(argumentsInJSON, "fundCode").String()
    if strings.TrimSpace(code) == "" {
        return "", fmt.Errorf("参数fundCode不能为空")
    }

    client := resty.New()
    // 1) 估算净值：天天基金 js 接口
    var est data.FundNetUnitValue
    resp1, err := client.SetTimeout(15*time.Second).R().
        SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36").
        SetHeader("Referer", "https://fund.eastmoney.com/").
        SetQueryParam("rt", strconv.FormatInt(time.Now().UnixMilli(), 10)).
        Get(fmt.Sprintf("https://fundgz.1234567.com.cn/js/%s.js", code))
    if err != nil {
        return "", err
    }
    if resp1.StatusCode() == 200 {
        body := string(resp1.Body())
        if strings.Contains(body, "jsonpgz") {
            body = strings.TrimPrefix(body, "jsonpgz(")
            body = strings.TrimSuffix(body, ");")
            if e := json.Unmarshal([]byte(body), &est); e != nil {
                // 容错：若解析失败，不中断流程
                est = data.FundNetUnitValue{}
            }
        }
    }

    // 2) 单位净值：新浪接口（GB18030 编码）
    name := ""
    dwjz := ""
    jzrq := ""
    resp2, err := client.SetTimeout(15*time.Second).R().
        SetHeader("Host", "hq.sinajs.cn").
        SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36 Edg/119.0.0.0").
        SetHeader("Referer", "https://finance.sina.com.cn").
        Get(fmt.Sprintf("http://hq.sinajs.cn/rn=%d&list=f_%s", time.Now().UnixMilli(), code))
    if err == nil && resp2.StatusCode() == 200 {
        dataStr := string(data.GB18030ToUTF8(resp2.Body()))
        parts := strings.Split(dataStr, "=")
        if len(parts) >= 2 {
            vals := strings.Split(strings.Trim(parts[1], "\"\n"), ",")
            if len(vals) >= 5 {
                name = vals[0]
                // 单位净值可能为浮点，做一次清洗避免科学计数或多余空格
                if v, e := convertor.ToFloat(vals[1]); e == nil {
                    dwjz = fmt.Sprintf("%v", v)
                } else {
                    dwjz = vals[1]
                }
                jzrq = vals[4]
            }
        }
    }

    // 3) 汇总表格
    row := fundNetValuesRow{
        Name:   ifNotEmpty(name, est.Name),
        Code:   code,
        Dwjz:   ifNotEmpty(dwjz, est.Dwjz),
        Jzrq:   jzrq,
        Gsz:    est.Gsz,
        Gszzl:  est.Gszzl,
        Gztime: est.Gztime,
    }
    return util.MarkdownTableWithTitle("基金净值数据("+code+")", []fundNetValuesRow{row}), nil
}

func ifNotEmpty(a, b string) string {
    if strings.TrimSpace(a) != "" {
        return a
    }
    return b
}