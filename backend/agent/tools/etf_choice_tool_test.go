package tools

import (
    "context"
    "testing"
    "go-stock/backend/db"
)

// 运行筛选并打印结果，分别执行A/B两套条件
func Test_ChoiceETFByIndicators(t *testing.T) {
    db.Init("../../data/stock.db")
    ctx := context.Background()
    tool := GetChoiceETFByIndicatorsTool()
    // 方案A：近1月≤3%，近3月≤8%，规模≥50亿，估算排序，前5
    wordsA := "筛选 T+0 指数ETF，近1月绝对值≤3%，近3月绝对值≤8%，规模≥50亿，按盘中估算涨跌幅排序，前5"
    resA, err := tool.InvokableRun(ctx, `{"words":"`+wordsA+`"}`)
    if err != nil {
        t.Logf("Plan A error: %v", err)
    }
    t.Log("Plan A Result:\n" + resA)

    // 方案B：近1月≤2%，近3月≤6%，规模≥30亿，估算排序，前5
    wordsB := "筛选 T+0 指数ETF，近1月绝对值≤2%，近3月绝对值≤6%，规模≥30亿，按盘中估算涨跌幅排序，前5"
    resB, err := tool.InvokableRun(ctx, `{"words":"`+wordsB+`"}`)
    if err != nil {
        t.Logf("Plan B error: %v", err)
    }
    t.Log("Plan B Result:\n" + resB)
}

// 使用用户提供的“T+1指数ETF”条件执行筛选并打印结果
func Test_ChoiceETF_T1_UserWords(t *testing.T) {
    db.Init("../../data/stock.db")
    ctx := context.Background()
    tool := GetChoiceETFByIndicatorsTool()
    words := "筛选 T+1 指数ETF，近1月绝对值≤2%，近3月绝对值≤6%，规模≥20亿，按盘中估算涨跌幅排序，前5"
    res, err := tool.InvokableRun(ctx, `{"words":"`+words+`"}`)
    if err != nil {
        t.Logf("T+1 UserWords error: %v", err)
    }
    t.Log("T+1 UserWords Result:\n" + res)
}

// 使用“近1月≤4%、近3月≤10%、规模>=0，前5”的条件执行筛选
func Test_ChoiceETF_UserWords_ScaleZero(t *testing.T) {
    db.Init("../../data/stock.db")
    ctx := context.Background()
    tool := GetChoiceETFByIndicatorsTool()
    words := "近1月≤4%、近3月≤10%、规模>=0，前5"
    res, err := tool.InvokableRun(ctx, `{"words":"`+words+`"}`)
    if err != nil {
        t.Logf("ScaleZero error: %v", err)
    }
    t.Log("ScaleZero Result:\n" + res)
}