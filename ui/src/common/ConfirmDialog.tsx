import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import React from 'react';

interface IProps {
    title: string;
    text: string;
    fClose: VoidFunction;
    fOnSubmit: () => void | Promise<void>;
}

export default function ConfirmDialog({title, text, fClose, fOnSubmit}: IProps) {
    const submitAndClose = async () => {
        try {
            await fOnSubmit();
        } catch (error) {
            console.error('ConfirmDialog submit failed:', error);
        } finally {
            fClose();
        }
    };
    return (
        <Dialog
            open={true}
            onClose={fClose}
            aria-labelledby="form-dialog-title"
            className="confirm-dialog">
            <DialogTitle id="form-dialog-title">{title}</DialogTitle>
            <DialogContent>
                <DialogContentText>{text}</DialogContentText>
            </DialogContent>
            <DialogActions>
                <Button onClick={fClose} color="secondary" variant="contained" className="cancel">
                    No
                </Button>
                <Button
                    onClick={submitAndClose}
                    autoFocus
                    color="primary"
                    variant="contained"
                    className="confirm">
                    Yes
                </Button>
            </DialogActions>
        </Dialog>
    );
}
