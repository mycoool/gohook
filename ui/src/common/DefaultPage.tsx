import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import {useTheme} from '@mui/material/styles';
import React, {FC} from 'react';

interface IProps {
    title: string;
    rightControl?: React.ReactNode;
    maxWidth?: number;
    centerTitle?: boolean;
    children?: React.ReactNode;
}

const DefaultPage: FC<IProps> = ({
    title,
    rightControl,
    maxWidth = 700,
    centerTitle = false,
    children,
}) => {
    const theme = useTheme();

    return (
        <main style={{margin: '0 auto', maxWidth}}>
            <Grid container spacing={2}>
                <Grid
                    size={12}
                    style={{
                        display: 'flex',
                        flexWrap: 'wrap',
                        alignItems: 'center',
                        marginBottom: '16px',
                    }}>
                    <Typography
                        variant="h4"
                        style={{
                            flex: 1,
                            textAlign: centerTitle ? 'center' : 'left',
                            fontWeight: 'normal',
                            color: theme.palette.text.primary,
                            margin: 0,
                        }}>
                        {title}
                    </Typography>
                    {rightControl}
                </Grid>
                {children}
            </Grid>
        </main>
    );
};
export default DefaultPage;
