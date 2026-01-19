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
import useTranslation from '../i18n/useTranslation';

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
    forceOptionLabel,
    forceOptionDescription,
    warningText,
}: IProps) {
    const {t} = useTranslation();
    const [forceEnabled, setForceEnabled] = useState(false);
    const resolvedForceLabel = forceOptionLabel ?? t('confirmDialog.forceOptionLabel');
    const resolvedForceDescription =
        forceOptionDescription ?? t('confirmDialog.forceOptionDescription');
    const resolvedWarningText = warningText ?? t('confirmDialog.warningText');

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
                                <strong>{resolvedForceLabel}</strong>
                                <br />
                                <span style={{fontSize: '0.875rem', color: '#666'}}>
                                    {resolvedForceDescription}
                                </span>
                            </span>
                        }
                    />
                </Box>

                {forceEnabled && (
                    <Alert severity="warning" sx={{mt: 1}}>
                        {resolvedWarningText}
                    </Alert>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={fClose} color="secondary" variant="contained" className="cancel">
                    {t('common.cancel')}
                </Button>
                <Button
                    onClick={submitAndClose}
                    autoFocus
                    color="primary"
                    variant="contained"
                    className="confirm">
                    {t('common.confirm')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
