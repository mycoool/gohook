import Grid from '@material-ui/core/Grid';
import Typography from '@material-ui/core/Typography';
import React, {FC} from 'react';

interface IProps {
    title: string;
    rightControl?: React.ReactNode;
    maxWidth?: number;
    centerTitle?: boolean;
}

const DefaultPage: FC<IProps> = ({title, rightControl, maxWidth = 700, centerTitle = false, children}) => (
    <main style={{margin: '0 auto', maxWidth}}>
        <Grid container spacing={4}>
            <Grid item xs={12} style={{display: 'flex', flexWrap: 'wrap'}}>
                <Typography variant="h4" style={{flex: 1, textAlign: centerTitle ? 'center' : 'left'}}>
                    {title}
                </Typography>
                {rightControl}
            </Grid>
            {children}
        </Grid>
    </main>
);
export default DefaultPage;
