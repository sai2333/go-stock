package tools

import (
    "context"
    "testing"
)

// 仅做最小验证：直接调用工具的运行方法，确保无panic且返回字符串
func Test_GetFundTools_Minimal(t *testing.T) {
    ctx := context.Background()

    // SearchFundByKey
    s := GetSearchFundByKeyTool()
    if _, err := s.InvokableRun(ctx, `{"key":"ETF"}`); err != nil {
        t.Logf("SearchFundByKey error: %v", err)
    }

    // GetFundBasic
    b := GetFundBasicTool()
    if _, err := b.InvokableRun(ctx, `{"fundCode":"510630"}`); err != nil {
        t.Logf("GetFundBasic error: %v", err)
    }

    // GetFundNetValues
    n := GetFundNetValuesTool()
    if _, err := n.InvokableRun(ctx, `{"fundCode":"510630"}`); err != nil {
        t.Logf("GetFundNetValues error: %v", err)
    }
}