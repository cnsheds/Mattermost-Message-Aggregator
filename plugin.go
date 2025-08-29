package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

const (
	POST_TYPE_AGGREGATED = "custom_aggregated"
)

// Plugin 插件结构体
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock 同步配置更新
	configurationLock sync.RWMutex

	// configuration 插件配置
	configuration *configuration
}

// configuration 插件配置结构
type configuration struct {
	TriggerWords        string
	MaxLookbackTime     int
	MaxLookbackMessages int
	RejectMessage       string
}

// Clone 深拷贝配置
func (c *configuration) Clone() *configuration {
	var clone = *c
	return &clone
}

// getConfiguration 获取当前配置
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration.Clone()
}

// OnActivate 插件激活时调用
func (p *Plugin) OnActivate() error {
	return nil
}

// OnConfigurationChange 配置变更时调用
func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(configuration)

	// 从配置中加载
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return fmt.Errorf("failed to load plugin configuration: %w", err)
	}

	// 设置默认值
	if configuration.TriggerWords == "" {
		configuration.TriggerWords = "收到,已收到,确认"
	}
	if configuration.MaxLookbackMessages == 0 {
		configuration.MaxLookbackMessages = 3
	}
	if configuration.MaxLookbackTime == 0 {
		configuration.MaxLookbackTime = 6
	}

	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	p.configuration = configuration
	return nil
}

// MessageWillBePosted 消息发布前的钩子
func (p *Plugin) dis_MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	// 忽略机器人消息
	if post.Props["from_bot"] == "true" {
		return post, ""
	}

	config := p.getConfiguration()
	triggerWords := strings.Split(config.TriggerWords, ",")

	post.Message = strings.TrimSpace(post.Message)
	// 检查消息是否包含触发词
	messageText := strings.TrimSpace(post.Message)
	isTriggerMessage := false

	for _, word := range triggerWords {
		word = strings.TrimSpace(word)
		if messageText == word {
			isTriggerMessage = true
			break
		}
	}

	if !isTriggerMessage {
		return post, ""
	}

	// 获取用户信息
	user, err := p.API.GetUser(post.UserId)
	if err != nil {
		p.API.LogError("Failed to get user", "error", err)
		return post, ""
	}

	displayName := user.Nickname
	if displayName == "" {
		displayName = user.Username
	}
	// 查找最近的相同消息
	recentPost := p.findRecentSimilarPost(post.ChannelId, messageText, "", config.MaxLookbackMessages)
	if recentPost != nil {
		if recentPost.Type == POST_TYPE_AGGREGATED {
			// 更新现有聚合消息的sender_names
			p.updateAggregatedPostSenderNames(recentPost, displayName)
		} else {
			// 创建聚合消息
			p.createAggregatedPost(recentPost, displayName, messageText)

			// 删除原消息
			if err := p.API.DeletePost(recentPost.Id); err != nil {
				p.API.LogError("Failed to delete original post", "error", err)
			}
		}

		// 发送系统提示给用户
		p.sendSystemEphemeralPost(post.UserId, post.ChannelId, config.RejectMessage)

		// 返回nil阻止当前消息发布
		return nil, config.RejectMessage
	}
	return post, ""
}

// sendSystemEphemeralPost 发送系统临时消息
func (p *Plugin) sendSystemEphemeralPost(userId, channelId, message string) {
	// 如果消息为空，则不发送
	if strings.TrimSpace(message) == "" {
		return
	}

	systemPost := &model.Post{
		UserId:    userId,
		ChannelId: channelId,
		Message:   message,
		Type:      "system_ephemeral",
	}
	p.API.SendEphemeralPost(userId, systemPost)
}

// MessageHasBeenPosted 消息发布后的钩子
func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	// 忽略机器人消息
	if post.Props["from_bot"] == "true" {
		return
	}

	config := p.getConfiguration()
	triggerWords := strings.Split(config.TriggerWords, ",")

	post.Message = strings.TrimSpace(post.Message)
	// 检查消息是否包含触发词
	messageText := strings.TrimSpace(post.Message)
	isTriggerMessage := false

	for _, word := range triggerWords {
		word = strings.TrimSpace(word)
		if messageText == word {
			isTriggerMessage = true
			break
		}
	}

	if !isTriggerMessage {
		return
	}

	// 获取用户信息
	user, err := p.API.GetUser(post.UserId)
	if err != nil {
		p.API.LogError("Failed to get user", "error", err)
		return
	}

	displayName := user.Nickname
	if displayName == "" {
		displayName = user.Username
	}

	// 查找最近的相同消息
	recentPost := p.findRecentSimilarPost(post.ChannelId, messageText, post.Id, config.MaxLookbackMessages+1)
	if recentPost != nil {
		if recentPost.Type == POST_TYPE_AGGREGATED {
			// 更新现有聚合消息的sender_names
			p.updateAggregatedPostSenderNames(recentPost, displayName)
		} else {
			// 创建聚合消息
			p.createAggregatedPost(recentPost, displayName, messageText)

			// 删除原消息
			if err := p.API.DeletePost(recentPost.Id); err != nil {
				p.API.LogError("Failed to delete original post", "error", err)
			}
		}

		// 发送系统提示给用户
		p.sendSystemEphemeralPost(post.UserId, post.ChannelId, config.RejectMessage)

		// 删除当前发布的消息
		if err := p.API.DeletePost(post.Id); err != nil {
			p.API.LogError("Failed to delete current post", "error", err)
		}
	}
}

// findRecentSimilarPost 查找最近的相似消息
func (p *Plugin) findRecentSimilarPost(channelId, messageText string, currentPostId string, maxLookback int) *model.Post {
	// 获取最近的消息
	postList, err := p.API.GetPostsForChannel(channelId, 0, maxLookback)
	if err != nil {
		p.API.LogError("Failed to get posts", "error", err)
		return nil
	}

	config := p.getConfiguration()
	// 按时间顺序检查消息
	for _, postId := range postList.Order {
		if postId == currentPostId {
			continue
		}
		post := postList.Posts[postId]

		// 检查消息时间是否超过6小时
		if (time.Now().Unix() - post.UpdateAt/1000) > int64(config.MaxLookbackTime*60*60) {
			break
		}

		// 优先检查聚合消息
		if post.Type == POST_TYPE_AGGREGATED {
			// 检查聚合消息是否与原始消息相同
			if strings.TrimSpace(post.Message) == messageText {
				return post
			}
		}

		// 检查是否是完全相同的消息
		if strings.TrimSpace(post.Message) == messageText {
			return post
		}
	}

	return nil
}

// createAggregatedPost 创建聚合消息
func (p *Plugin) createAggregatedPost(originalPost *model.Post, newUsername, originalMessage string) *model.Post {
	// 获取原消息的发布者信息
	originalUser, err := p.API.GetUser(originalPost.UserId)
	if err != nil {
		p.API.LogError("Failed to get original user", "error", err)
		return nil
	}

	originalDisplayName := originalUser.Nickname
	if originalDisplayName == "" {
		originalDisplayName = originalUser.Username
	}

	// 创建新的聚合消息
	aggregatedPost := &model.Post{
		ChannelId: originalPost.ChannelId,
		Message:   originalMessage,
		Type:      POST_TYPE_AGGREGATED,
	}

	// 在props中添加sender_names数组
	senderNames := []string{originalDisplayName}
	if newUsername != originalDisplayName {
		senderNames = append(senderNames, newUsername)
	}
	aggregatedPost.AddProp("sender_names", senderNames)
	aggregatedPost.AddProp("from_bot", "true")

	// 创建机器人用户来发布消息
	botUserID := p.getBotUserID()
	if botUserID != "" {
		aggregatedPost.UserId = botUserID
	}

	// 发布聚合消息
	if _, err := p.API.CreatePost(aggregatedPost); err != nil {
		p.API.LogError("Failed to create aggregated post", "error", err)
		return nil
	}

	return aggregatedPost
}

// updateAggregatedPostSenderNames 更新聚合消息的sender_names
func (p *Plugin) updateAggregatedPostSenderNames(post *model.Post, newUsername string) {
	// 获取现有的sender_names
	var senderNames []string
	if names, ok := post.Props["sender_names"].([]interface{}); ok {
		for _, name := range names {
			if str, ok := name.(string); ok {
				senderNames = append(senderNames, str)
			}
		}
	}

	// 检查用户是否已经在列表中
	for _, existingName := range senderNames {
		if existingName == newUsername {
			// 用户已存在，不需要添加
			return
		}
	}

	// 添加新用户名
	senderNames = append(senderNames, newUsername)
	post.AddProp("sender_names", senderNames)

	// 更新消息
	if _, err := p.API.UpdatePost(post); err != nil {
		p.API.LogError("Failed to update aggregated post sender_names", "error", err)
	}
}

// getBotUserID 获取机器人用户ID
func (p *Plugin) getBotUserID() string {
	// 获取系统机器人用户
	botUser, err := p.API.GetUserByUsername("bot")
	if err != nil {
		// 如果没有找到bot用户，尝试获取系统用户
		systemUser, err := p.API.GetUserByUsername("system")
		if err != nil {
			p.API.LogError("Failed to get bot or system user", "error", err)
			return ""
		}
		return systemUser.Id
	}
	return botUser.Id
}

func main() {
	plugin.ClientMain(&Plugin{})
}
