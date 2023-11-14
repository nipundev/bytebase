package tidb

// Framework code is generated by the generator.

import (
	"fmt"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*DisallowLimitAdvisor)(nil)
	_ ast.Visitor     = (*disallowLimitChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLDisallowLimit, &DisallowLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLDisallowLimit, &DisallowLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLDisallowLimit, &DisallowLimitAdvisor{})
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLDisallowLimit, &DisallowLimitAdvisor{})
}

// DisallowLimitAdvisor is the advisor checking for no LIMIT clause in INSERT/UPDATE statement.
type DisallowLimitAdvisor struct {
}

// Check checks for no LIMIT clause in INSERT/UPDATE statement.
func (*DisallowLimitAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &disallowLimitChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.text = stmt.Text()
		checker.line = stmt.OriginTextPosition()
		(stmt).Accept(checker)
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

type disallowLimitChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
}

// Enter implements the ast.Visitor interface.
func (checker *disallowLimitChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := advisor.Ok
	switch node := in.(type) {
	case *ast.UpdateStmt:
		if node.Limit != nil {
			code = advisor.UpdateUseLimit
		}
	case *ast.DeleteStmt:
		if node.Limit != nil {
			code = advisor.DeleteUseLimit
		}
	case *ast.InsertStmt:
		if useLimit(node) {
			code = advisor.InsertUseLimit
		}
	}

	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    code,
			Title:   checker.title,
			Content: fmt.Sprintf("LIMIT clause is forbidden in INSERT, UPDATE and DELETE statement, but \"%s\" uses", checker.text),
			Line:    checker.line,
		})
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*disallowLimitChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func useLimit(node *ast.InsertStmt) bool {
	if node.Select != nil {
		switch stmt := node.Select.(type) {
		case *ast.SelectStmt:
			return stmt.Limit != nil
		case *ast.SetOprStmt:
			return stmt.Limit != nil
		}
	}
	return false
}
