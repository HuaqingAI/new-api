export function formatClassicAlertDepartments(snapshot) {
  if (!Array.isArray(snapshot) || snapshot.length === 0) {
    return '未归属';
  }
  return snapshot
    .map((item) => `${item.department_name} (#${item.department_id})`)
    .join(', ');
}
