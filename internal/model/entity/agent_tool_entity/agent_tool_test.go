package agent_tool_entity_test

import (
	"context"
	"testing"

	"github.com/cago-frame/cago/pkg/utils/httputils"
	. "github.com/smartystreets/goconvey/convey"

	"agentre/internal/model/entity/agent_tool_entity"
	"agentre/internal/pkg/code"
)

func TestAgentTool_IsActive(t *testing.T) {
	Convey("AgentTool.IsActive", t, func() {
		Convey("nil receiver returns false", func() {
			var a *agent_tool_entity.AgentTool
			So(a.IsActive(), ShouldBeFalse)
		})

		Convey("status=ACTIVE returns true", func() {
			a := &agent_tool_entity.AgentTool{Status: 1}
			So(a.IsActive(), ShouldBeTrue)
		})

		Convey("status=DELETE returns false", func() {
			a := &agent_tool_entity.AgentTool{Status: 2}
			So(a.IsActive(), ShouldBeFalse)
		})

		Convey("zero status returns false", func() {
			a := &agent_tool_entity.AgentTool{Status: 0}
			So(a.IsActive(), ShouldBeFalse)
		})
	})
}

func TestAgentTool_Check(t *testing.T) {
	Convey("AgentTool.Check", t, func() {
		ctx := context.Background()

		Convey("nil receiver returns AgentToolNotFound", func() {
			var a *agent_tool_entity.AgentTool
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.AgentToolNotFound)
		})

		Convey("empty name returns InvalidParameter", func() {
			a := &agent_tool_entity.AgentTool{
				Name:         "   ",
				ExecutorType: "builtin",
				ParamSchema:  "{}",
			}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.InvalidParameter)
		})

		Convey("invalid executor_type returns AgentToolInvalidExecutorType", func() {
			a := &agent_tool_entity.AgentTool{
				Name:         "my_tool",
				ExecutorType: "unknown",
				ParamSchema:  "{}",
			}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.AgentToolInvalidExecutorType)
		})

		Convey("invalid param_schema JSON returns AgentToolInvalidParamSchema", func() {
			a := &agent_tool_entity.AgentTool{
				Name:         "my_tool",
				ExecutorType: "builtin",
				ParamSchema:  "{invalid",
			}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.AgentToolInvalidParamSchema)
		})

		Convey("invalid executor_conf JSON returns AgentToolInvalidExecutorConf", func() {
			a := &agent_tool_entity.AgentTool{
				Name:         "my_tool",
				ExecutorType: "http",
				ParamSchema:  "{}",
				ExecutorConf: "{bad",
			}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.AgentToolInvalidExecutorConf)
		})

		Convey("valid builtin tool passes", func() {
			a := &agent_tool_entity.AgentTool{
				Name:         "read_file",
				ExecutorType: "builtin",
				ParamSchema:  `{"type":"object","properties":{"path":{"type":"string"}}}`,
				ExecutorConf: "{}",
			}
			err := a.Check(ctx)
			So(err, ShouldBeNil)
		})

		Convey("valid http tool passes", func() {
			a := &agent_tool_entity.AgentTool{
				Name:         "webhook",
				ExecutorType: "http",
				ParamSchema:  "{}",
				ExecutorConf: `{"url":"https://example.com/hook"}`,
			}
			err := a.Check(ctx)
			So(err, ShouldBeNil)
		})

		Convey("empty executor_conf passes", func() {
			a := &agent_tool_entity.AgentTool{
				Name:         "my_tool",
				ExecutorType: "builtin",
				ParamSchema:  "{}",
				ExecutorConf: "",
			}
			err := a.Check(ctx)
			So(err, ShouldBeNil)
		})

		Convey("default executor_conf passes", func() {
			a := &agent_tool_entity.AgentTool{
				Name:         "my_tool",
				ExecutorType: "builtin",
				ParamSchema:  "{}",
				ExecutorConf: "{}",
			}
			err := a.Check(ctx)
			So(err, ShouldBeNil)
		})

		Convey("valid mcp tool passes", func() {
			a := &agent_tool_entity.AgentTool{
				Name:         "mcp_search",
				ExecutorType: "mcp",
				ParamSchema:  `{"type":"object"}`,
				ExecutorConf: `{"server":"localhost:3000"}`,
			}
			err := a.Check(ctx)
			So(err, ShouldBeNil)
		})
	})
}
