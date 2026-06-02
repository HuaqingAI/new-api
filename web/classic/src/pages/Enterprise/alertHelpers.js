export function formatClassicAlertDepartments(snapshot) {
  if (!Array.isArray(snapshot) || snapshot.length === 0) {
    return '未归属';
  }
  return snapshot
    .map((item) => `${item.department_name} (#${item.department_id})`)
    .join(', ');
}

export function formatClassicEnterpriseUserPrimary(input = {}) {
  const displayName = input.display_name?.trim();
  if (displayName) return displayName;
  const username = input.username?.trim();
  if (username) return username;
  if (input.user_id != null) return `#${input.user_id}`;
  return '-';
}

export function formatClassicEnterpriseUserSecondary(
  input = {},
  t = (value) => value,
) {
  const displayName = input.display_name?.trim();
  const username = input.username?.trim();
  const userId = input.user_id;

  if (displayName && username && displayName !== username) {
    if (userId != null) {
      return `${username} · ${t('用户 ID')} #${userId}`;
    }
    return username;
  }
  if (username) return `${t('用户名')}: ${username}`;
  if (userId != null) return `${t('用户 ID')} #${userId}`;
  return '-';
}
