// MessageMerger Webapp Plugin
import AggregatedPost from './components/AggregatedPost.jsx';

// 定义自定义帖子类型常量，必须和后端保持一致！
const POST_TYPE_AGGREGATED = 'custom_aggregated';

class MPlugin {
    initialize(registry, store) {
        console.log('MessageMerger webapp plugin initializing...');
        registry.registerPostTypeComponent(POST_TYPE_AGGREGATED, AggregatedPost);
    }

    uninitialize() {
        console.log('MessageMerger webapp plugin uninitialize');
    }
}

window.registerPlugin('message_merger', new MPlugin());
