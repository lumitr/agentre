package agent_memory_entity_test

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"agentre/internal/model/entity/agent_memory_entity"
)

func TestAgentMemory_Check(t *testing.T) {
	Convey("AgentMemory.Check", t, func() {
		ctx := context.Background()

		Convey("nil receiver returns error", func() {
			var a *agent_memory_entity.AgentMemory
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("empty agent_id returns error", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 0, Scope: "user", Category: "preference", Content: "test"}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("invalid scope returns error", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "invalid", Category: "preference", Content: "test"}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("invalid category returns error", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "user", Category: "invalid", Content: "test"}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("empty content returns error", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "user", Category: "preference", Content: ""}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("session scope without session_id returns error", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "session", Category: "summary", Content: "test", SessionID: 0}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
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
