import Fab from '@mui/material/Fab';
import KeyboardArrowUp from '@mui/icons-material/KeyboardArrowUp';
import React, {Component} from 'react';

class ScrollUpButton extends Component {
    private observer?: MutationObserver;
    
    state = {
        display: 'none',
        opacity: 0,
        bottomOffset: 80, // 默认底部偏移
    };
    componentDidMount() {
        window.addEventListener('scroll', this.scrollHandler);
        this.adjustPosition();
        
        // 监听DOM变化，当实时消息面板显示/隐藏时调整位置
        this.observer = new MutationObserver(() => {
            this.adjustPosition();
        });
        
        // 监听body的子元素变化
        this.observer.observe(document.body, {
            childList: true,
            subtree: true,
            attributes: true,
            attributeFilter: ['data-realtime-messages']
        });
    }

    componentWillUnmount() {
        window.removeEventListener('scroll', this.scrollHandler);
        if (this.observer) {
            this.observer.disconnect();
        }
    }

    // 动态调整位置，避免与其他固定元素重叠
    adjustPosition = () => {
        // 检查是否有实时消息面板展开
        const realtimeMessagesPanel = document.querySelector('[data-realtime-messages="expanded"]');
        const realtimeMessagesButton = document.querySelector('[data-realtime-messages="button"]');
        
        let bottomOffset = 80; // 默认位置
        
        if (realtimeMessagesPanel) {
            // 如果实时消息面板展开，位置需要更高一些
            bottomOffset = 520; // 面板高度大约500px + 一些间距
        } else if (realtimeMessagesButton) {
            // 如果只有按钮，保持当前位置
            bottomOffset = 80;
        }
        
        if (this.state.bottomOffset !== bottomOffset) {
            this.setState({bottomOffset});
        }
    };

    scrollHandler = () => {
        const currentScrollPos = window.pageYOffset;
        const opacity = Math.min(currentScrollPos / 500, 1);
        const nextState = {display: currentScrollPos > 0 ? 'inherit' : 'none', opacity};
        if (this.state.display !== nextState.display || this.state.opacity !== nextState.opacity) {
            this.setState(nextState);
        }
        
        // 每次滚动时也检查位置
        this.adjustPosition();
    };

    public render() {
        return (
            <Fab
                color="primary"
                size="small" // 与实时消息按钮保持一致的大小
                style={{
                    position: 'fixed',
                    bottom: `${this.state.bottomOffset}px`, // 动态调整底部偏移
                    right: '16px', // 与实时消息按钮保持一致的右边距
                    zIndex: 100000,
                    display: this.state.display,
                    opacity: this.state.opacity,
                    transition: 'bottom 0.3s ease', // 添加过渡动画
                }}
                onClick={this.scrollUp}>
                <KeyboardArrowUp />
            </Fab>
        );
    }

    private scrollUp = () => window.scrollTo(0, 0);
}

export default ScrollUpButton;
