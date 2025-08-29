import React from 'react';

// 为组件定义一些内联样式，使其看起来像图片中的样子
const style = {
    // 整个卡片的外部容器
    container: {
        width: '100%',
    },
    // 消息内容部分
    content: {
        marginBottom: '8px', // 与下方用户列表的间距
        lineHeight: '1.5',
    },
    // 下方灰色用户列表的容器
    userListContainer: {
        display: 'flex',
        alignItems: 'center',
        backgroundColor: '#f5f5f5', // 浅灰色背景
        borderRadius: '4px',
        padding: '8px 12px',
        color: '#9e00faff', // 较深的灰色文字
    },
    // 人名列表的样式
    userNames: {
        flex: 1, // 占据剩余空间
        whiteSpace: 'nowrap',
        overflow: 'hidden',
        textOverflow: 'ellipsis', // 如果人名太长，用省略号显示
    },
};

/**
 * 聚合消息的自定义帖子组件
 * @param {object} props - 组件属性
 * @param {object} props.post - Mattermost 的 Post 对象
 */
const AggregatedPost = ({ post }) => {
    // 从 post.props 中安全地解构出后端插件设置的数据
    const aggregatedMessage = post.message;
    const senderNames = post.props?.sender_names || [];

    // 如果没有发送者列表，可以不渲染下方的灰色区域
    const hasSenders = senderNames.length > 0;

    return (
        <div style={style.container}>
            {/* 1. 消息内容 */}
            <div style={style.content}>
                {aggregatedMessage}
            </div>

            {/* 2. 发送者列表 (仅当有发送者时显示) */}
            {hasSenders && (
                <div style={style.userListContainer}>
                    <span style={style.userNames}>
                        {/* 使用 '、' 将人名数组连接成一个字符串 */}
                        {senderNames.join('、')}
                    </span>
                </div>
            )}
        </div>
    );
};

export default AggregatedPost;