package agent_memory_svc

// MemoryItem 记忆条目 DTO。
type MemoryItem struct {
	ID         int64  `json:"id"`
	AgentID    int64  `json:"agentId"`
	Scope      string `json:"scope"`
	SessionID  int64  `json:"sessionId"`
	Category   string `json:"category"`
	Key        string `json:"key"`
	Content    string `json:"content"`
	Source     string `json:"source"`
	Createtime int64  `json:"createtime"`
	Updatetime int64  `json:"updatetime"`
}

type CreateMemoryRequest struct {
	AgentID   int64  `json:"agentId" binding:"required"`
	Scope     string `json:"scope" binding:"required"`
	SessionID int64  `json:"sessionId"`
	Category  string `json:"category" binding:"required"`
	Key       string `json:"key"`
	Content   string `json:"content" binding:"required"`
}

type CreateMemoryResponse struct {
	Item *MemoryItem `json:"item"`
}

type UpdateMemoryRequest struct {
	ID       int64  `json:"id" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Category string `json:"category"`
	Key      string `json:"key"`
}

type UpdateMemoryResponse struct {
	Item *MemoryItem `json:"item"`
}

type DeleteMemoryRequest struct {
	ID int64 `json:"id" binding:"required"`
}

type DeleteMemoryResponse struct{}

type ListMemoryRequest struct {
	AgentID int64  `json:"agentId" binding:"required"`
	Scope   string `json:"scope"`
}

type ListMemoryResponse struct {
	Items []*MemoryItem `json:"items"`
}
