type UserDisplayInput = {
  displayName?: string | null
  username?: string | null
  userId?: number | null
}

export function formatEnterpriseUserPrimary(input: UserDisplayInput) {
  const displayName = input.displayName?.trim()
  if (displayName) return displayName
  const username = input.username?.trim()
  if (username) return username
  if (input.userId != null) return `#${input.userId}`
  return '-'
}

export function formatEnterpriseUserSecondary(
  input: UserDisplayInput,
  t: (key: string) => string
) {
  const displayName = input.displayName?.trim()
  const username = input.username?.trim()

  if (displayName && username && displayName !== username) {
    return `${username} · ${t('User ID')} #${input.userId ?? '-'}`
  }
  if (username) return `${t('Username')}: ${username}`
  if (input.userId != null) return `${t('User ID')} #${input.userId}`
  return '-'
}
