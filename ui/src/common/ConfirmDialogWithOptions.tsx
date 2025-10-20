import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import FormControlLabel from '@mui/material/FormControlLabel';
import Checkbox from '@mui/material/Checkbox';
import Box from '@mui/material/Box';
import Alert from '@mui/material/Alert';
import React, {useState} from 'react';

interface IProps {
    title: string;
    text: string;
    fClose: VoidFunction;
    fOnSubmit: (force: boolean) => void;
    forceOptionLabel?: string;
    forceOptionDescription?: string;
    warningText?: string;
}

export default function ConfirmDialogWithOptions({
    title,
    text,
    fClose,
    fOnSubmit,
    forceOptionLabel = '强制同步（丢弃本地修改）',
    forceOptionDescription = '启用此选项将会强制覆盖本地修改，确保与远程仓库同步',
    warningText = '⚠️ 注意：强制同步会永久丢弃所有未提交的本地修改',
}: IProps) {
    const [forceEnabled, setForceEnabled] = useState(false);

    const submitAndClose = () => {
        fOnSubmit(forceEnabled);
        fClose();
    };

    return (
        <Dialog
            open={true}
            onClose={fClose}
            aria-labelledby="form-dialog-title"
            className="confirm-dialog-with-options"
            maxWidth="sm"
            fullWidth>
            <DialogTitle id="form-dialog-title">{title}</DialogTitle>
            <DialogContent>
                <DialogContentText>{text}</DialogContentText>

                <Box mt={2} mb={2}>
                    <FormControlLabel
                        control={
                            <Checkbox
                                checked={forceEnabled}
                                onChange={(e) => setForceEnabled(e.target.checked)}
                                color="primary"
                            />
                        }
                        label={
                            <span>
                                <strong>{forceOptionLabel}</strong>
                                <br />
                                <span style={{fontSize: '0.875rem', color: '#666'}}>
                                    {forceOptionDescription}
                                </span>
                            </span>
                        }
                    />
                </Box>

                {forceEnabled && (
                    <Alert severity="warning" sx={{mt: 1}}>
                        {warningText}
                    </Alert>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={fClose} color="secondary" variant="contained" className="cancel">
                    取消
                </Button>
                <Button
                    onClick={submitAndClose}
                    autoFocus
                    color="primary"
                    variant="contained"
                    className="confirm">
                    确认
                </Button>
            </DialogActions>
        </Dialog>
    );
}
