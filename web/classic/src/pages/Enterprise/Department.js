/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Empty,
  Form,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  getDepartmentMembers,
  getUserDepartments,
} from '../../services/enterprise';
import { showError } from '../../helpers';
import {
  formatClassicEnterpriseUserPrimary,
  formatClassicEnterpriseUserSecondary,
} from './alertHelpers';

const statusMap = {
  1: { color: 'green', key: '启用' },
  2: { color: 'grey', key: '停用' },
  3: { color: 'grey', key: '已离开' },
  4: { color: 'orange', key: '待处理' },
};

export default function EnterpriseDepartment() {
  const { t } = useTranslation();
  const [userDepartments, setUserDepartments] = useState([]);
  const [departmentMembers, setDepartmentMembers] = useState([]);
  const [isUnassigned, setIsUnassigned] = useState(false);
  const [loading, setLoading] = useState(false);

  async function loadUserDepartments(values) {
    if (!values.user_id) return;
    setLoading(true);
    try {
      const res = await getUserDepartments(values.user_id);
      if (!res.success) {
        showError(res.message);
        return;
      }
      setUserDepartments(res.data?.items || []);
      setIsUnassigned(Boolean(res.data?.is_unassigned));
    } finally {
      setLoading(false);
    }
  }

  async function loadDepartmentMembers(values) {
    if (!values.department_id) return;
    setLoading(true);
    try {
      const res = await getDepartmentMembers(values.department_id);
      if (!res.success) {
        showError(res.message);
        return;
      }
      setDepartmentMembers(res.data?.items || []);
    } finally {
      setLoading(false);
    }
  }

  const statusRender = (status) => {
    const config = statusMap[status] || statusMap[4];
    return <Tag color={config.color}>{t(config.key)}</Tag>;
  };

  return (
    <div className='dashboard-container'>
      <Typography.Title heading={4}>{t('企业组织')}</Typography.Title>
      <Space vertical align='start' style={{ width: '100%' }}>
        <Card title={t('用户所属部门')}>
          <Form layout='horizontal' onSubmit={loadUserDepartments}>
            <Form.Input
              field='user_id'
              label={t('用户 ID')}
              style={{ width: 220 }}
            />
            <Button htmlType='submit' loading={loading}>
              {t('查询')}
            </Button>
          </Form>
          {isUnassigned ? (
            <Empty
              title={t('未归属')}
              description={t('该用户没有部门成员关系')}
            />
          ) : (
            <Table
              pagination={false}
              dataSource={userDepartments}
              columns={[
                { title: t('部门'), dataIndex: 'department_name' },
                { title: t('部门 ID'), dataIndex: 'department_id' },
                {
                  title: t('状态'),
                  dataIndex: 'status',
                  render: statusRender,
                },
                { title: t('来源'), dataIndex: 'external_source' },
              ]}
            />
          )}
        </Card>

        <Card title={t('部门成员')}>
          <Form layout='horizontal' onSubmit={loadDepartmentMembers}>
            <Form.Input
              field='department_id'
              label={t('部门 ID')}
              style={{ width: 220 }}
            />
            <Button htmlType='submit' loading={loading}>
              {t('查询')}
            </Button>
          </Form>
          <Table
            pagination={false}
            dataSource={departmentMembers}
            columns={[
              {
                title: t('用户'),
                dataIndex: 'username',
                render: (_, record) => (
                  <div>
                    <div>{formatClassicEnterpriseUserPrimary(record)}</div>
                    <Typography.Text type='tertiary' size='small'>
                      {formatClassicEnterpriseUserSecondary(record, t)}
                    </Typography.Text>
                  </div>
                ),
              },
              { title: t('用户 ID'), dataIndex: 'user_id' },
              {
                title: t('状态'),
                dataIndex: 'status',
                render: statusRender,
              },
              { title: t('来源'), dataIndex: 'external_source' },
            ]}
          />
        </Card>
      </Space>
    </div>
  );
}
