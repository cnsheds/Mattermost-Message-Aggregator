package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
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
		configuration.MaxLookbackMessages = 10
	}

	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	p.configuration = configuration

	return nil
}

// MessageWillBePosted 消息发布前的钩子
func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
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
	recentPost := p.findRecentSimilarPost(post.ChannelId, messageText, config.MaxLookbackMessages)
	if recentPost != nil {
		// 更新现有消息
		p.updatePostWithUser(recentPost, displayName, messageText)

		// 发送系统提示给用户
		systemPost := &model.Post{
			UserId:    post.UserId,
			ChannelId: post.ChannelId,
			Message:   config.RejectMessage,
			Type:      "system_ephemeral",
		}
		p.API.SendEphemeralPost(post.UserId, systemPost)

		// 返回nil阻止当前消息发布
		return nil, config.RejectMessage
	}
	return post, ""
}

// findRecentSimilarPost 查找最近的相似消息
func (p *Plugin) findRecentSimilarPost(channelId, messageText string, maxLookback int) *model.Post {
	// 获取最近的消息
	postList, err := p.API.GetPostsForChannel(channelId, 0, maxLookback+1)
	if err != nil {
		p.API.LogError("Failed to get posts", "error", err)
		return nil
	}

	// 编译正则表达式来匹配聚合消息格式
	pattern := fmt.Sprintf(`^%s\s*--\s*.+`, regexp.QuoteMeta(messageText))
	regex, re_err := regexp.Compile(pattern)
	if re_err != nil {
		p.API.LogError("Failed to compile regex", "error", re_err.Error())
		return nil
	}

	// 按时间顺序检查消息
	for _, postId := range postList.Order {
		post := postList.Posts[postId]

		// 检查是否是聚合格式的相同消息
		if regex.MatchString(post.Message) {
			return post
		}

		// 检查是否是完全相同的消息
		if strings.TrimSpace(post.Message) == messageText {
			return post
		}
	}

	return nil
}

// updatePostWithUser 更新帖子添加用户名
func (p *Plugin) updatePostWithUser(post *model.Post, username, originalMessage string) {
	// 解析现有的聚合消息
	parts := strings.Split(post.Message, " -- ")
	if len(parts) >= 2 {
		// 检查用户是否已经在列表中
		userList := strings.Split(parts[1], ", ")
		for _, existingUser := range userList {
			if strings.TrimSpace(existingUser) == username {
				// 用户已存在，不需要添加
				return
			}
		}

		// 添加新用户
		updatedMessage := fmt.Sprintf("%s -- %s, %s", parts[0], parts[1], username)
		post.Message = updatedMessage
	} else {
		// 不是预期的格式，重新创建
		post.Message = fmt.Sprintf("%s -- %s", originalMessage, username)
	}

	// 更新消息
	if _, err := p.API.UpdatePost(post); err != nil {
		p.API.LogError("Failed to update aggregated post", "error", err)
	}
}

func main() {
	plugin.ClientMain(&Plugin{})
}
