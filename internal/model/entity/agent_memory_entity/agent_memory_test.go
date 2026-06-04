package agent_memory_entity_test

import (
	"context"
	"testing"

	"github.com/cago-frame/cago/pkg/utils/httputils"
	. "github.com/smartystreets/goconvey/convey"

	"agentre/internal/model/entity/agent_memory_entity"
	"agentre/internal/pkg/code"
)

func TestAgentMemory_IsActive(t *testing.T) {
	Convey("AgentMemory.IsActive", t, func() {
		Convey("nil receiver returns false", func() {
			var a *agent_memory_entity.AgentMemory
			So(a.IsActive(), ShouldBeFalse)
		})

		Convey("status=ACTIVE returns true", func() {
			a := &agent_memory_entity.AgentMemory{Status: 1}
			So(a.IsActive(), ShouldBeTrue)
		})

		Convey("status=DELETE returns false", func() {
			a := &agent_memory_entity.AgentMemory{Status: 2}
			So(a.IsActive(), ShouldBeFalse)
		})

		Convey("zero status returns false", func() {
			a := &agent_memory_entity.AgentMemory{Status: 0}
			So(a.IsActive(), ShouldBeFalse)
		})
	})
}

func TestAgentMemory_Check(t *testing.T) {
	Convey("AgentMemory.Check", t, func() {
		ctx := context.Background()

		Convey("nil receiver returns AgentMemoryNotFound", func() {
			var a *agent_memory_entity.AgentMemory
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.AgentMemoryNotFound)
		})

		Convey("zero agent_id returns InvalidParameter", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 0, Scope: "user", Category: "preference", Content: "test"}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.InvalidParameter)
		})

		Convey("negative agent_id returns InvalidParameter", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: -1, Scope: "user", Category: "preference", Content: "test"}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.InvalidParameter)
		})

		Convey("invalid scope returns AgentMemoryInvalidScope", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "invalid", Category: "preference", Content: "test"}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.AgentMemoryInvalidScope)
		})

		Convey("invalid category returns AgentMemoryInvalidCategory", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "user", Category: "invalid", Content: "test"}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.AgentMemoryInvalidCategory)
		})

		Convey("empty content returns AgentMemoryEmptyContent", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "user", Category: "preference", Content: ""}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.AgentMemoryEmptyContent)
		})

		Convey("session scope without session_id returns AgentMemorySessionRequired", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "session", Category: "summary", Content: "test", SessionID: 0}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
			var httpErr *httputils.Error
			So(err, ShouldHaveSameTypeAs, httpErr)
			httpErr = err.(*httputils.Error)
			So(httpErr.Code, ShouldEqual, code.AgentMemorySessionRequired)
		})

		Convey("valid user preference passes", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "user", Category: "preference", Content: "likes TypeScript", Key: "language"}
			err := a.Check(ctx)
			So(err, ShouldBeNil)
		})

		Convey("valid session summary passes", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "session", Category: "summary", Content: "discussed API design", SessionID: 42}
			err := a.Check(ctx)
			So(err, ShouldBeNil)
		})
	})
}
