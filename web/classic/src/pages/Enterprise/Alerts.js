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
import { Button, Card, Empty, Form, Space, Table, Typography } from '@douyinfe/semi-ui';
import { getAlertEvents } from '../../services/enterprise';
import { showError } from '../../helpers';
import { formatClassicAlertDepartments } from './alertHelpers';

export default function EnterpriseAlerts() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [events, setEvents] = useState([]);
  const [pagination, setPagination] = useState({ total: 0, page: 1, page_size: 20 });

  async function loadEvents(values) {
    setLoading(true);
    try {
      const res = await getAlertEvents({
        department_id: values.department_id || undefined,
        user_id: values.user_id || undefined,
        username: values.username || undefined,
        model_name: values.model_name || undefined,
        risk_type: values.risk_type || undefined,
        page: values.page || 1,
        page_size: values.page_size || 20,
      });
      if (!res.success) {
        showError(res.message);
        return;
      }
      setEvents(res.data?.items || []);
      setPagination({
        total: res.data?.total || 0,
        page: res.data?.page || 1,
        page_size: res.data?.page_size || 20,
      });
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className='dashboard-container'>
      <Typography.Title heading={4}>{t('企业风险事件')}</Typography.Title>
      <Space vertical align='start' style={{ width: '100%' }}>
        <Card title={t('查询')}>
          <Form layout='horizontal' onSubmit={loadEvents}>
            <Form.Input field='department_id' label={t('部门 ID')} style={{ width: 220 }} />
            <Form.Input field='user_id' label={t('用户 ID')} style={{ width: 220 }} />
            <Form.Input field='username' label={t('用户')} style={{ width: 220 }} />
            <Form.Input field='model_name' label={t('模型')} style={{ width: 220 }} />
            <Form.Input field='risk_type' label={t('风险类型')} style={{ width: 220 }} />
            <Form.Input field='page_size' label={t('每页条数')} style={{ width: 220 }} initValue='20' />
            <Button htmlType='submit' loading={loading}>
              {t('查询')}
            </Button>
          </Form>
        </Card>

        <Card title={t('企业风险事件列表')} style={{ width: '100%' }}>
          {events.length === 0 ? (
            <Empty title={t('暂无风险事件')} description={t('请调整筛选条件后重试')} />
          ) : (
            <Table
              pagination={false}
              dataSource={events}
              columns={[
                { title: t('发生时间'), dataIndex: 'created_at' },
                {
                  title: t('部门'),
                  dataIndex: 'department_snapshot',
                  render: formatClassicAlertDepartments,
                },
                { title: t('用户'), dataIndex: 'username' },
                { title: t('请求 ID'), dataIndex: 'request_id' },
                { title: t('模型'), dataIndex: 'model_name' },
                { title: t('风险类型'), dataIndex: 'risk_type' },
                { title: t('动作结果'), dataIndex: 'action_result' },
                { title: t('摘要'), dataIndex: 'summary' },
              ]}
            />
          )}
          <div style={{ marginTop: 12, color: 'var(--semi-color-text-2)' }}>
            {t('第 {{page}} 页，共 {{total}} 条', { page: pagination.page, total: pagination.total })}
          </div>
        </Card>
      </Space>
    </div>
  );
}
