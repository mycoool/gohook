import { styled } from '@mui/material/styles';
import Paper from '@mui/material/Paper';
import * as React from 'react';

const StyledPaper = styled(Paper)({
    padding: 16,
});

interface IProps {
    style?: React.CSSProperties;
    children?: React.ReactNode;
}

const Container: React.FC<IProps> = ({ children, style }) => (
    <StyledPaper elevation={6} style={style}>
        {children}
    </StyledPaper>
);

export default Container;
