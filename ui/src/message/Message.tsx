import IconButton from '@mui/material/IconButton';
import {styled} from '@mui/material/styles';
import Typography from '@mui/material/Typography';
import Delete from '@mui/icons-material/Delete';
import React from 'react';
import TimeAgo from 'react-timeago';
import Container from '../common/Container';
import * as config from '../config';
import {Markdown} from '../common/Markdown';
import {RenderMode, contentType} from './extras';
import {IMessageExtras} from '../types';

const MessageWrapper = styled('div')(({theme}) => ({
    padding: 12,
}));

const Header = styled('div')({
    display: 'flex',
    flexWrap: 'wrap',
    marginBottom: 0,
});

const HeaderTitle = styled(Typography)({
    flex: 1,
});

const TrashButton = styled(IconButton)({
    marginTop: -15,
    marginRight: -15,
});

const MessageContentWrapper = styled('div')({
    width: '100%',
    maxWidth: 585,
});

const Image = styled('img')(({theme}) => ({
    marginRight: 15,
    [theme.breakpoints.down('lg')]: {
        width: 32,
        height: 32,
    },
}));

const Date = styled(Typography)(({theme}) => ({
    [theme.breakpoints.down('lg')]: {
        order: 1,
        flexBasis: '100%',
        opacity: 0.7,
    },
}));

const ImageWrapper = styled('div')({
    display: 'flex',
});

const PlainContent = styled('span')({
    whiteSpace: 'pre-wrap',
});

const Content = styled('div')({
    wordBreak: 'break-all',
    '& p': {
        margin: 0,
    },
    '& a': {
        color: '#ff7f50',
    },
    '& pre': {
        overflow: 'auto',
    },
    '& img': {
        maxWidth: '100%',
    },
});

interface IProps {
    title: string;
    image?: string;
    date: string;
    content: string;
    priority: number;
    fDelete: VoidFunction;
    extras?: IMessageExtras;
    height: (height: number) => void;
}

const priorityColor = (priority: number) => {
    if (priority >= 4 && priority <= 7) {
        return 'rgba(230, 126, 34, 0.7)';
    } else if (priority > 7) {
        return '#e74c3c';
    } else {
        return 'transparent';
    }
};

class Message extends React.PureComponent<IProps> {
    private node: HTMLDivElement | null = null;

    public componentDidMount = () =>
        this.props.height(this.node ? this.node.getBoundingClientRect().height : 0);

    private renderContent = () => {
        const content = this.props.content;
        switch (contentType(this.props.extras)) {
            case RenderMode.Markdown:
                return <Markdown>{content}</Markdown>;
            case RenderMode.Plain:
            default:
                return <PlainContent>{content}</PlainContent>;
        }
    };

    public render(): React.ReactNode {
        const {fDelete, title, date, image, priority} = this.props;

        return (
            <MessageWrapper
                className="message"
                ref={(ref) => {
                    this.node = ref;
                }}>
                <Container
                    style={{
                        display: 'flex',
                        borderLeftColor: priorityColor(priority),
                        borderLeftWidth: 6,
                        borderLeftStyle: 'solid',
                    }}>
                    <ImageWrapper>
                        {image !== null ? (
                            <Image
                                src={config.get('url') + image}
                                alt="app logo"
                                width="70"
                                height="70"
                            />
                        ) : null}
                    </ImageWrapper>
                    <MessageContentWrapper>
                        <Header>
                            <HeaderTitle className="title" variant="h5">
                                {title}
                            </HeaderTitle>
                            <Date variant="body1">
                                <TimeAgo date={date} />
                            </Date>
                            <TrashButton onClick={fDelete} className="delete" size="large">
                                <Delete />
                            </TrashButton>
                        </Header>
                        <Content className="content">{this.renderContent()}</Content>
                    </MessageContentWrapper>
                </Container>
            </MessageWrapper>
        );
    }
}

export default Message;
