import Grid from '@material-ui/core/Grid';
import IconButton from '@material-ui/core/IconButton';
import Paper from '@material-ui/core/Paper';
import {withStyles, WithStyles} from '@material-ui/core/styles';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Delete from '@material-ui/icons/Delete';
import Edit from '@material-ui/icons/Edit';
import React, {Component} from 'react';
import ConfirmDialog from '../common/ConfirmDialog';
import DefaultPage from '../common/DefaultPage';
import Button from '@material-ui/core/Button';
import AddEditDialog from './AddEditUserDialog';
import {observer} from 'mobx-react';
import {observable} from 'mobx';
import {inject, Stores} from '../inject';
import {IUser} from '../types';
import useTranslation from '../i18n/useTranslation';

const styles = () => ({
    wrapper: {
        margin: '0 auto',
        maxWidth: 700,
    },
});

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
                <IconButton onClick={fEdit} className="edit" title={t('user.editUser')}>
                    <Edit />
                </IconButton>
                <IconButton onClick={fDelete} className="delete" title={t('user.deleteUser')}>
                    <Delete />
                </IconButton>
            </TableCell>
        </TableRow>
    );
};

@observer
class Users extends Component<WithStyles<'wrapper'> & Stores<'userStore'>> {
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
            <Grid item xs={12}>
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

export default withStyles(styles)(inject('userStore')(Users));
