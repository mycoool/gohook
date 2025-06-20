import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';

import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Delete from '@mui/icons-material/Delete';
import Edit from '@mui/icons-material/Edit';
import React, {Component} from 'react';
import ConfirmDialog from '../common/ConfirmDialog';
import DefaultPage from '../common/DefaultPage';
import Button from '@mui/material/Button';
import AddEditDialog from './AddEditUserDialog';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {IUser} from '../types';
import useTranslation from '../i18n/useTranslation';

interface IRowProps {
    name: string;
    admin: boolean;
    fDelete: VoidFunction;
    fEdit: VoidFunction;
}

const UserRow: React.FC<IRowProps> = ({name, admin, fDelete, fEdit}) => {
    const {t} = useTranslation();

    return (
        <TableRow>
            <TableCell>{name}</TableCell>
            <TableCell>{admin ? t('user.admin') : t('user.normalUser')}</TableCell>
            <TableCell align="right" padding="none">
                <IconButton onClick={fEdit} className="edit" title={t('user.editUser')} size="large">
                    <Edit />
                </IconButton>
                <IconButton
                    onClick={fDelete}
                    className="delete"
                    title={t('user.deleteUser')}
                    size="large">
                    <Delete />
                </IconButton>
            </TableCell>
        </TableRow>
    );
};

@observer
class Users extends Component<Stores<'userStore'>> {
    @observable
    private createDialog = false;
    @observable
    private deleteId: number | false = false;
    @observable
    private editId: number | false = false;

    public componentDidMount = () => this.props.userStore.refresh();

    public render() {
        const {userStore} = this.props;
        const users = userStore.getItems();

        return (
            <UsersContainer
                users={users}
                createDialog={this.createDialog}
                deleteId={this.deleteId}
                editId={this.editId}
                onCreateUser={() => (this.createDialog = true)}
                onEditUser={(id) => (this.editId = id)}
                onDeleteUser={(id) => (this.deleteId = id)}
                onCloseCreateDialog={() => (this.createDialog = false)}
                onCloseEditDialog={() => (this.editId = false)}
                onCloseDeleteDialog={() => (this.deleteId = false)}
                onSubmitCreate={userStore.create}
                onSubmitEdit={(id) => userStore.update.bind(this, id)}
                onSubmitDelete={(id) => userStore.remove(id)}
                userStore={userStore}
            />
        );
    }
}

// 分离容器组件以使用Hook
const UsersContainer: React.FC<{
    users: IUser[];
    createDialog: boolean;
    deleteId: number | false;
    editId: number | false;
    onCreateUser: () => void;
    onEditUser: (id: number) => void;
    onDeleteUser: (id: number) => void;
    onCloseCreateDialog: () => void;
    onCloseEditDialog: () => void;
    onCloseDeleteDialog: () => void;
    onSubmitCreate: (name: string, password: string, admin: boolean) => void;
    onSubmitEdit: (id: number) => (name: string, password: string, admin: boolean) => void;
    onSubmitDelete: (id: number) => void;
    userStore: {getByID: (id: number) => IUser};
}> = ({
    users,
    createDialog,
    deleteId,
    editId,
    onCreateUser,
    onEditUser,
    onDeleteUser,
    onCloseCreateDialog,
    onCloseEditDialog,
    onCloseDeleteDialog,
    onSubmitCreate,
    onSubmitEdit,
    onSubmitDelete,
    userStore,
}) => {
    const {t} = useTranslation();

    return (
        <DefaultPage
            title={t('user.title')}
            rightControl={
                <Button id="create-user" variant="contained" color="primary" onClick={onCreateUser}>
                    {t('user.addUser')}
                </Button>
            }>
            <Grid size={12}>
                <Paper elevation={6} style={{overflowX: 'auto'}}>
                    <Table id="user-table">
                        <TableHead>
                            <TableRow style={{textAlign: 'center'}}>
                                <TableCell>{t('user.username')}</TableCell>
                                <TableCell>{t('user.role')}</TableCell>
                                <TableCell />
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {users.map((user: IUser) => (
                                <UserRow
                                    key={user.id}
                                    name={user.name}
                                    admin={user.admin}
                                    fDelete={() => onDeleteUser(user.id)}
                                    fEdit={() => onEditUser(user.id)}
                                />
                            ))}
                        </TableBody>
                    </Table>
                </Paper>
            </Grid>
            {createDialog && (
                <AddEditDialog fClose={onCloseCreateDialog} fOnSubmit={onSubmitCreate} />
            )}
            {editId !== false && (
                <AddEditDialog
                    fClose={onCloseEditDialog}
                    fOnSubmit={onSubmitEdit(editId)}
                    name={userStore.getByID(editId).name}
                    admin={userStore.getByID(editId).admin}
                    isEdit={true}
                />
            )}
            {deleteId !== false && (
                <ConfirmDialog
                    title={t('user.deleteUser')}
                    text={t('user.confirmDeleteText', {name: userStore.getByID(deleteId).name})}
                    fClose={onCloseDeleteDialog}
                    fOnSubmit={() => onSubmitDelete(deleteId)}
                />
            )}
        </DefaultPage>
    );
};

export default inject('userStore')(Users);
