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

import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Empty,
  Form,
  Space,
  Table,
  Tabs,
  Typography,
} from '@douyinfe/semi-ui';
import { getAlertEvents, getAlertRules, saveAlertRule } from '../../services/enterprise';
import { showError, showSuccess } from '../../helpers';
import { formatClassicAlertDepartments } from './alertHelpers';

function createEmptyRuleDraft() {
  return {
    id: undefined,
    name: '',
    enabled: true,
    risk_types: 'abuse\nsensitive_words',
    department_ids: '',
    email_receivers: '',
  };
}

function mapRuleToDraft(rule) {
  const email = (rule.channel_configs || []).find((item) => item.type === 'email');
  return {
    id: rule.id,
    name: rule.name || '',
    enabled: rule.enabled !== false,
    risk_types: (rule.risk_types || []).join('\n'),
    department_ids: (rule.department_ids || []).join(', '),
    email_receivers: (email?.receivers || []).join('\n'),
  };
}

function buildRulePayload(draft) {
  return {
    ...(draft.id ? { id: draft.id } : {}),
    name: draft.name,
    enabled: draft.enabled,
    risk_types: draft.risk_types
      .split(/[\n,]/g)
      .map((item) => item.trim())
      .filter(Boolean),
    department_ids: draft.department_ids
      .split(/[\n,]/g)
      .map((item) => Number.parseInt(item.trim(), 10))
      .filter((item) => Number.isFinite(item) && item > 0),
    channel_configs: [
      {
        type: 'email',
        enabled: true,
        receivers: draft.email_receivers
          .split(/[\n,]/g)
          .map((item) => item.trim())
          .filter(Boolean),
      },
      {
        type: 'webhook',
        enabled: false,
      },
    ],
  };
}

export default function EnterpriseAlerts() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [savingRule, setSavingRule] = useState(false);
  const [events, setEvents] = useState([]);
  const [rules, setRules] = useState([]);
  const [draft, setDraft] = useState(createEmptyRuleDraft());
  const [pagination, setPagination] = useState({ total: 0, page: 1, page_size: 20 });
  const [ruleFormApi, setRuleFormApi] = useState(null);

  async function loadEvents(values = {}) {
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

  async function loadRules() {
    try {
      const res = await getAlertRules();
      if (!res.success) {
        showError(res.message);
        return;
      }
      const items = res.data?.items || [];
      setRules(items);
      if (!draft.id && items.length > 0) {
        const nextDraft = mapRuleToDraft(items[0]);
        setDraft(nextDraft);
        ruleFormApi?.setValues(nextDraft);
      }
    } catch (error) {
      showError(error.message);
    }
  }

  async function handleSaveRule() {
    setSavingRule(true);
    try {
      const res = await saveAlertRule(buildRulePayload(draft));
      if (!res.success) {
        showError(res.message);
        return;
      }
      showSuccess(t('保存成功'));
      const item = res.data?.item;
      if (item) {
        setDraft(mapRuleToDraft(item));
      }
      await loadRules();
    } finally {
      setSavingRule(false);
    }
  }

  function applyDraft(nextDraft) {
    setDraft(nextDraft);
    ruleFormApi?.setValues(nextDraft);
  }

  useEffect(() => {
    void loadEvents();
    void loadRules();
  }, []);

  return (
    <div className='dashboard-container'>
      <Typography.Title heading={4}>{t('企业风险事件')}</Typography.Title>
      <Tabs type='line'>
        <Tabs.TabPane tab={t('企业风险事件列表')} itemKey='events'>
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
        </Tabs.TabPane>

        <Tabs.TabPane tab={t('告警规则')} itemKey='rules'>
          <Space vertical align='start' style={{ width: '100%' }}>
            <Card title={t('告警规则列表')} style={{ width: '100%' }}>
              {rules.length === 0 ? (
                <Empty title={t('暂无告警规则')} description={t('请先创建邮件告警规则')} />
              ) : (
                <Table
                  pagination={false}
                  dataSource={rules}
                  rowKey='id'
                  onRow={(record) => ({
                    onClick: () => applyDraft(mapRuleToDraft(record)),
                  })}
                  columns={[
                    { title: t('名称'), dataIndex: 'name' },
                    {
                      title: t('状态'),
                      dataIndex: 'enabled',
                      render: (value) => (value ? t('启用') : t('停用')),
                    },
                    {
                      title: t('风险类型'),
                      dataIndex: 'risk_types',
                      render: (value) => (value || []).join(', '),
                    },
                    {
                      title: t('部门范围'),
                      dataIndex: 'department_ids',
                      render: (value) =>
                        value && value.length ? value.join(', ') : t('全部部门'),
                    },
                  ]}
                />
              )}
            </Card>

            <Card title={draft.id ? t('编辑告警规则') : t('创建告警规则')} style={{ width: '100%' }}>
              <Form
                layout='vertical'
                initValues={draft}
                getFormApi={(api) => setRuleFormApi(api)}
                onValueChange={(values) => setDraft((prev) => ({ ...prev, ...values }))}
              >
                <Form.Input field='name' label={t('规则名称')} />
                <Form.Switch field='enabled' label={t('启用规则')} />
                <Form.TextArea
                  field='risk_types'
                  label={t('风险类型')}
                  placeholder={t('每行一个风险类型')}
                />
                <Form.TextArea
                  field='department_ids'
                  label={t('部门范围')}
                  placeholder={t('留空表示全部部门，多个部门 ID 用逗号或换行分隔')}
                />
                <Form.TextArea
                  field='email_receivers'
                  label={t('邮件接收人')}
                  placeholder={t('每行一个邮箱地址')}
                />
                <Space>
                  <Button theme='solid' loading={savingRule} onClick={handleSaveRule}>
                    {t('保存规则')}
                  </Button>
                  <Button onClick={() => applyDraft(createEmptyRuleDraft())}>
                    {t('新建规则')}
                  </Button>
                </Space>
              </Form>
            </Card>
          </Space>
        </Tabs.TabPane>
      </Tabs>
    </div>
  );
}
