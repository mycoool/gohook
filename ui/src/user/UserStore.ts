import {BaseStore} from '../common/BaseStore';
import axios from 'axios';
import * as config from '../config';
import {action} from 'mobx';
import {SnackReporter} from '../snack/SnackManager';
import {IUser} from '../types';

export interface IUserResponse {
    username: string;
    role: string;
}

export class UserStore extends BaseStore<IUser> {
    constructor(private readonly snack: SnackReporter, private readonly tokenProvider: () => string) {
        super();
    }

    protected requestItems = (): Promise<IUser[]> =>
        axios.get<IUserResponse[]>(`${config.get('url')}user`, {
            headers: {'X-GoHook-Key': this.tokenProvider()}
        }).then((response) => 
            // 转换响应数据为IUser格式
            response.data.map((user, index) => ({
                id: index + 1, // 使用索引作为临时ID
                name: user.username,
                username: user.username,
                admin: user.role === 'admin',
                role: user.role
            }))
        );

    protected requestDelete(id: number): Promise<void> {
        // 从当前项目中找到对应的用户名
        const user = this.getByID(id);
        if (!user || !user.username) {
            return Promise.reject(new Error('User not found'));
        }
        
        return axios
            .delete(`${config.get('url')}user/${user.username}`, {
                headers: {'X-GoHook-Key': this.tokenProvider()}
            })
            .then(() => this.snack('User deleted'));
    }

    @action
    public create = async (name: string, pass: string, admin: boolean) => {
        const role = admin ? 'admin' : 'user';
        await axios.post(`${config.get('url')}user`, {
            username: name,
            password: pass,
            role: role
        }, {
            headers: {'X-GoHook-Key': this.tokenProvider()}
        });
        await this.refresh();
        this.snack('User created');
    };

    @action
    public update = async (id: number, name: string, pass: string | null) => {
        // 由于后端没有提供用户更新API，这里暂时抛出错误
        // 如果需要更新用户，应该实现重置密码功能
        if (pass && pass.length > 0) {
            // 重置密码
            const user = this.getByID(id);
            if (!user || !user.username) {
                throw new Error('User not found');
            }
            
            await axios.post(`${config.get('url')}user/${user.username}/reset-password`, {
                newPassword: pass
            }, {
                headers: {'X-GoHook-Key': this.tokenProvider()}
            });
            this.snack('Password updated');
        } else {
            this.snack('User update functionality is limited. Only password reset is supported.');
        }
        await this.refresh();
    };

    @action
    public changePassword = async (oldPassword: string, newPassword: string) => {
        await axios.post(`${config.get('url')}user/password`, {
            oldPassword: oldPassword,
            newPassword: newPassword
        }, {
            headers: {'X-GoHook-Key': this.tokenProvider()}
        });
        this.snack('Password changed successfully');
    };
}
