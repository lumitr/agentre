package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migration202606040001 建 agent_memories 表。
func migration202606040001() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202606040001",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Exec(`CREATE TABLE IF NOT EXISTS agent_memories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	agent_id INTEGER NOT NULL,
	scope TEXT NOT NULL,
	session_id INTEGER NOT NULL DEFAULT 0,
	category TEXT NOT NULL,
	key TEXT NOT NULL DEFAULT '',
	content TEXT NOT NULL,
	source TEXT NOT NULL DEFAULT 'manual',
	status INTEGER NOT NULL DEFAULT 1,
	createtime INTEGER NOT NULL DEFAULT 0,
	updatetime INTEGER NOT NULL DEFAULT 0
)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_memories_agent_scope ON agent_memories(agent_id, scope)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_memories_session ON agent_memories(session_id) WHERE scope = 'session'`).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`DROP TABLE IF EXISTS agent_memories`).Error
		},
	}
}
