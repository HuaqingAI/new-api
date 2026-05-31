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

import { API } from '../helpers';

export async function getUserDepartments(userId) {
  const res = await API.get(`/api/enterprise/users/${userId}/departments`);
  return res.data;
}

export async function getDepartmentMembers(departmentId) {
  const res = await API.get(
    `/api/enterprise/departments/${departmentId}/members`,
  );
  return res.data;
}

export async function getAlertEvents(params = {}) {
  const res = await API.get('/api/enterprise/alerts/events', {
    params,
  });
  return res.data;
}

export async function getDepartmentRiskSummary(params = {}) {
  const res = await API.get('/api/enterprise/alerts/department-summary', {
    params,
  });
  return res.data;
}

export async function getAlertRules(params = {}) {
  const res = await API.get('/api/enterprise/alerts/rules', {
    params,
  });
  return res.data;
}

export async function getAlertDeliveries(params = {}) {
  const res = await API.get('/api/enterprise/alerts/deliveries', {
    params,
  });
  return res.data;
}

export async function resendAlertDelivery(id, params = {}) {
  const res = await API.post(`/api/enterprise/alerts/deliveries/${id}/resend`, null, {
    params,
  });
  return res.data;
}

export async function saveAlertRule(data) {
  const res = await API.put('/api/enterprise/alerts/rules', data);
  return res.data;
}
