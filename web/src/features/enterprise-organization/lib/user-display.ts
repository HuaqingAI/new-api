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
  const userId = input.userId

  if (displayName && username && displayName !== username) {
    if (userId != null) {
      return `${username} · ${t('User ID')} #${userId}`
    }
    return username
  }
  if (username) return `${t('Username')}: ${username}`
  if (userId != null) return `${t('User ID')} #${userId}`
  return '-'
}
