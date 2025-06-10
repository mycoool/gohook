import Grid from '@material-ui/core/Grid';
import IconButton from '@material-ui/core/IconButton';
import Paper from '@material-ui/core/Paper';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Delete from '@material-ui/icons/Delete';
import Edit from '@material-ui/icons/Edit';
import CloudUpload from '@material-ui/icons/CloudUpload';
import React, {ChangeEvent, Component, SFC} from 'react';
import ConfirmDialog from '../common/ConfirmDialog';
import DefaultPage from '../common/DefaultPage';
import Button from '@material-ui/core/Button';
import CopyableSecret from '../common/CopyableSecret';
import AddApplicationDialog from './AddApplicationDialog';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import * as config from '../config';
import UpdateDialog from './UpdateApplicationDialog';
import {IApplication} from '../types';
import {LastUsedCell} from '../common/LastUsedCell';
import useTranslation from '../i18n/useTranslation';

@observer
class Applications extends Component<Stores<'appStore'>> {
    @observable
    private deleteId: number | false = false;
    @observable
    private updateId: number | false = false;
    @observable
    private createDialog = false;

    private uploadId = -1;
    private upload: HTMLInputElement | null = null;

    public componentDidMount = () => this.props.appStore.refresh();

    public render() {
        const {appStore} = this.props;
        const apps = appStore.getItems();
        
        return (
            <ApplicationsContainer
                apps={apps}
                createDialog={this.createDialog}
                deleteId={this.deleteId}
                updateId={this.updateId}
                onCreateApp={() => this.createDialog = true}
                onEditApp={(id: number) => this.updateId = id}
                onDeleteApp={(id: number) => this.deleteId = id}
                onUploadImage={this.uploadImage}
                onCloseCreateDialog={() => this.createDialog = false}
                onCloseEditDialog={() => this.updateId = false}
                onCloseDeleteDialog={() => this.deleteId = false}
                onSubmitCreate={appStore.create}
                onSubmitEdit={(id: number, name: string, description: string, defaultPriority: number) =>
                    appStore.update(id, name, description, defaultPriority)
                        }
                onSubmitDelete={(id: number) => appStore.remove(id)}
                appStore={appStore}
                upload={this.upload}
                onUploadRef={(ref: HTMLInputElement | null) => this.upload = ref}
                onUploadChange={this.onUploadImage}
                    />
        );
    }

    private uploadImage = (id: number) => {
        this.uploadId = id;
        if (this.upload) {
            this.upload.click();
        }
    };

    private onUploadImage = (e: ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) {
            return;
        }
        if (['image/png', 'image/jpeg', 'image/gif'].indexOf(file.type) !== -1) {
            this.props.appStore.uploadImage(this.uploadId, file);
        } else {
            alert('Uploaded file must be of type png, jpeg or gif.');
        }
    };
}

interface IRowProps {
    name: string;
    value: string;
    noDelete: boolean;
    description: string;
    defaultPriority: number;
    lastUsed: string | null;
    fUpload: VoidFunction;
    image: string;
    fDelete: VoidFunction;
    fEdit: VoidFunction;
}

const Row: SFC<IRowProps> = observer(
    ({
        name,
        value,
        noDelete,
        description,
        defaultPriority,
        lastUsed,
        fDelete,
        fUpload,
        image,
        fEdit,
    }) => (
        <TableRow>
            <TableCell padding="default">
                <div style={{display: 'flex'}}>
                    <img src={config.get('url') + image} alt="app logo" width="40" height="40" />
                    <IconButton onClick={fUpload} style={{height: 40}}>
                        <CloudUpload />
                    </IconButton>
                </div>
            </TableCell>
            <TableCell>{name}</TableCell>
            <TableCell>
                <CopyableSecret value={value} style={{display: 'flex', alignItems: 'center'}} />
            </TableCell>
            <TableCell>{description}</TableCell>
            <TableCell>{defaultPriority}</TableCell>
            <TableCell>
                <LastUsedCell lastUsed={lastUsed} />
            </TableCell>
            <TableCell align="right" padding="none">
                <IconButton onClick={fEdit} className="edit">
                    <Edit />
                </IconButton>
            </TableCell>
            <TableCell align="right" padding="none">
                <IconButton onClick={fDelete} className="delete" disabled={noDelete}>
                    <Delete />
                </IconButton>
            </TableCell>
        </TableRow>
    )
);

// 分离容器组件以使用Hook
const ApplicationsContainer: React.FC<{
    apps: IApplication[];
    createDialog: boolean;
    deleteId: number | false;
    updateId: number | false;
    onCreateApp: () => void;
    onEditApp: (id: number) => void;
    onDeleteApp: (id: number) => void;
    onUploadImage: (id: number) => void;
    onCloseCreateDialog: () => void;
    onCloseEditDialog: () => void;
    onCloseDeleteDialog: () => void;
    onSubmitCreate: (name: string, description: string, defaultPriority: number) => void;
    onSubmitEdit: (id: number, name: string, description: string, defaultPriority: number) => void;
    onSubmitDelete: (id: number) => void;
    appStore: { getByID: (id: number) => IApplication };
    upload: HTMLInputElement | null;
    onUploadRef: (ref: HTMLInputElement | null) => void;
    onUploadChange: (e: ChangeEvent<HTMLInputElement>) => void;
}> = ({
    apps,
    createDialog,
    deleteId,
    updateId,
    onCreateApp,
    onEditApp,
    onDeleteApp,
    onUploadImage,
    onCloseCreateDialog,
    onCloseEditDialog,
    onCloseDeleteDialog,
    onSubmitCreate,
    onSubmitEdit,
    onSubmitDelete,
    appStore,
    onUploadRef,
    onUploadChange
}) => {
    const { t } = useTranslation();

    return (
        <DefaultPage
            title={t('application.title')}
            rightControl={
                <Button
                    id="create-app"
                    variant="contained"
                    color="primary"
                    onClick={onCreateApp}>
                    {t('application.createApplication')}
                </Button>
            }
            maxWidth={1000}>
            <Grid item xs={12}>
                <Paper elevation={6} style={{overflowX: 'auto'}}>
                    <Table id="app-table">
                        <TableHead>
                            <TableRow>
                                <TableCell padding="checkbox" style={{width: 80}} />
                                <TableCell>{t('application.name')}</TableCell>
                                <TableCell>{t('application.token')}</TableCell>
                                <TableCell>{t('application.description')}</TableCell>
                                <TableCell>{t('application.priority')}</TableCell>
                                <TableCell>{t('application.lastUsed')}</TableCell>
                                <TableCell />
                                <TableCell />
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {apps.map((app: IApplication) => (
                                <ApplicationRow
                                    key={app.id}
                                    description={app.description}
                                    defaultPriority={app.defaultPriority}
                                    image={app.image}
                                    name={app.name}
                                    value={app.token}
                                    lastUsed={app.lastUsed}
                                    fUpload={() => onUploadImage(app.id)}
                                    fDelete={() => onDeleteApp(app.id)}
                                    fEdit={() => onEditApp(app.id)}
                                    noDelete={app.internal}
                                />
                            ))}
                        </TableBody>
                    </Table>
                    <input
                        ref={onUploadRef}
                        type="file"
                        style={{display: 'none'}}
                        onChange={onUploadChange}
                    />
                </Paper>
            </Grid>
            {createDialog && (
                <AddApplicationDialog
                    fClose={onCloseCreateDialog}
                    fOnSubmit={onSubmitCreate}
                />
            )}
            {updateId !== false && (
                <UpdateDialog
                    fClose={onCloseEditDialog}
                    fOnSubmit={(name: string, description: string, defaultPriority: number) =>
                        onSubmitEdit(updateId, name, description, defaultPriority)
                    }
                    initialDescription={appStore.getByID(updateId).description}
                    initialName={appStore.getByID(updateId).name}
                    initialDefaultPriority={appStore.getByID(updateId).defaultPriority}
                />
            )}
            {deleteId !== false && (
                <ConfirmDialog
                    title={t('application.confirmDelete')}
                    text={t('application.confirmDeleteText', { name: appStore.getByID(deleteId).name })}
                    fClose={onCloseDeleteDialog}
                    fOnSubmit={() => onSubmitDelete(deleteId)}
                />
            )}
        </DefaultPage>
    );
};

// 重命名Row组件以避免冲突
const ApplicationRow: SFC<IRowProps> = observer(
    ({
        name,
        value,
        noDelete,
        description,
        defaultPriority,
        lastUsed,
        fDelete,
        fUpload,
        image,
        fEdit,
    }) => {
        const { t } = useTranslation();
        
        return (
            <TableRow>
                <TableCell padding="default">
                    <div style={{display: 'flex'}}>
                        <img src={config.get('url') + image} alt="app logo" width="40" height="40" />
                        <IconButton onClick={fUpload} style={{height: 40}} title={t('application.uploadImage')}>
                            <CloudUpload />
                        </IconButton>
                    </div>
                </TableCell>
                <TableCell>{name}</TableCell>
                <TableCell>
                    <CopyableSecret value={value} style={{display: 'flex', alignItems: 'center'}} />
                </TableCell>
                <TableCell>{description}</TableCell>
                <TableCell>{defaultPriority}</TableCell>
                <TableCell>
                    <LastUsedCell lastUsed={lastUsed} />
                </TableCell>
                <TableCell align="right" padding="none">
                    <IconButton onClick={fEdit} className="edit" title={t('application.editApplication')}>
                        <Edit />
                    </IconButton>
                </TableCell>
                <TableCell align="right" padding="none">
                    <IconButton 
                        onClick={fDelete} 
                        className="delete" 
                        disabled={noDelete}
                        title={noDelete ? t('application.internal') : t('application.deleteApplication')}>
                        <Delete />
                    </IconButton>
                </TableCell>
            </TableRow>
        );
    }
);

export default inject('appStore')(Applications);
