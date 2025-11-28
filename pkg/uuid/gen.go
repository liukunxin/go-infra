package uuid

import (
	"github.com/liukunxin/go-infra/pkg/utils"
	"sync"

	"github.com/bwmarrin/snowflake"
	"github.com/gofrs/uuid"
	"github.com/sony/sonyflake"
)

type IDService struct {
	snowflakeNode *snowflake.Node
	sonyFlake     *sonyflake.Sonyflake
	nodeID        int64
}

var (
	instance *IDService
	once     sync.Once
)

func GetIDService() *IDService {
	once.Do(func() {
		nodeID := getNodeID() // 从环境变量或配置获取

		// 初始化snowflake
		sfNode, err := snowflake.NewNode(nodeID)
		if err != nil {
			panic("Snowflake初始化失败: " + err.Error())
		}

		// 初始化sonyflake（备用）
		sf := sonyflake.NewSonyflake(sonyflake.Settings{
			MachineID: func() (uint16, error) {
				return uint16(nodeID), nil
			},
		})

		instance = &IDService{
			snowflakeNode: sfNode,
			sonyFlake:     sf,
			nodeID:        nodeID,
		}
	})
	return instance
}

// GenerateUserID 生成用户ID（主要方案）
func (s *IDService) GenerateUserID() int64 {
	return s.snowflakeNode.Generate().Int64()
}

// GenerateUserIDFallback 生成用户ID（备用方案）
func (s *IDService) GenerateUserIDFallback() (uint64, error) {
	return s.sonyFlake.NextID()
}

// GenerateSessionID 生成会话ID
func (s *IDService) GenerateSessionID() string {
	// 优先使用UUID v7（时间有序）
	if id, err := uuid.NewV7(); err == nil {
		return id.String()
	}
	// 降级到UUID v4
	return uuid.Must(uuid.NewV4()).String()
}

// GenerateShortID 生成短ID（用于邀请码等）
func (s *IDService) GenerateShortID() string {
	id := s.snowflakeNode.Generate()
	return utils.Base58Encode(id.Int64())
}

// ParseID 从ID解析信息（调试用）
func (s *IDService) ParseID(id int64) snowflake.ID {
	return snowflake.ParseInt64(id)
}

// 获取节点ID
func getNodeID() int64 {
	// 从环境变量获取，确保不同实例不同ID
	// 或者使用IP地址最后一段，Pod名称等
	return 1 // 生产环境需要动态获取
}
