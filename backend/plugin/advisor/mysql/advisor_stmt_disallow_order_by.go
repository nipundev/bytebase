package mysql

// Framework code is generated by the generator.

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*DisallowOrderByAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLDisallowOrderBy, &DisallowOrderByAdvisor{})
}

// DisallowOrderByAdvisor is the advisor checking for no ORDER BY clause in DELETE/UPDATE statements.
type DisallowOrderByAdvisor struct {
}

// Check checks for no ORDER BY clause in DELETE/UPDATE statements.
func (*DisallowOrderByAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &disallowOrderByChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type disallowOrderByChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
}

func (checker *disallowOrderByChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterDeleteStatement is called when production deleteStatement is entered.
func (checker *disallowOrderByChecker) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if ctx.OrderClause() != nil && ctx.OrderClause().ORDER_SYMBOL() != nil {
		checker.handleOrderByClause(advisor.DeleteUseOrderBy, ctx.GetStart().GetLine())
	}
}

// EnterUpdateStatement is called when production updateStatement is entered.
func (checker *disallowOrderByChecker) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if ctx.OrderClause() != nil && ctx.OrderClause().ORDER_SYMBOL() != nil {
		checker.handleOrderByClause(advisor.UpdateUseOrderBy, ctx.GetStart().GetLine())
	}
}

func (checker *disallowOrderByChecker) handleOrderByClause(code advisor.Code, lineNumber int) {
	checker.adviceList = append(checker.adviceList, advisor.Advice{
		Status:  checker.level,
		Code:    code,
		Title:   checker.title,
		Content: fmt.Sprintf("ORDER BY clause is forbidden in DELETE and UPDATE statements, but \"%s\" uses", checker.text),
		Line:    checker.line + lineNumber,
	})
}
