import i18n from '@/lib/i18n'

const VALIDATION_MAP: Record<string, string> = {
  'must not be empty': 'validation.mustNotBeEmpty',
  'must be between 1 and 65535': 'validation.portRange',
  'must not be empty for password auth': 'validation.mustNotBeEmptyForPasswordAuth',
  'must not be empty for privateKey auth': 'validation.mustNotBeEmptyForPrivateKeyAuth',
  'must be provided for privateKey auth': 'validation.mustBeProvidedForPrivateKeyAuth',
  'must not be empty for inline source': 'validation.mustNotBeEmptyForInlineSource',
  'must not be empty for file source': 'validation.mustNotBeEmptyForFileSource',
  "must be 'inline' or 'file'": 'validation.mustBeInlineOrFile',
  "must be 'password', 'privateKey', or 'none'": 'validation.mustBePasswordPrivateKeyOrNone',
  'must be provided for dynamic mode': 'validation.mustBeProvidedForDynamicMode',
  'must not be empty for userpass auth': 'validation.mustNotBeEmptyForUserpassAuth',
  "must be 'none' or 'userpass'": 'validation.mustBeNoneOrUserpass',
  "must be 'local', 'remote', or 'dynamic'": 'validation.mustBeLocalRemoteOrDynamic',
  'must contain at least one SSH connection': 'validation.mustContainAtLeastOneSSH',
  'must contain at least one mapping': 'validation.mustContainAtLeastOneMapping',
  'must be unique within tunnel': 'validation.mustBeUniqueWithinTunnel',
  'must be greater than 0': 'validation.mustBeGreaterThan0',
  'must be greater than or equal to minMs': 'validation.mustBeGreaterThanOrEqualToMinMs',
  'must be greater than or equal to 1.0': 'validation.mustBeGreaterThanOrEqualTo1',
  'must be unique': 'validation.mustBeUnique',
}

const VALIDATION_PATTERNS: { pattern: RegExp; key: string }[] = [
  { pattern: /^must not be empty for (\w+) mode$/, key: 'validation.mustNotBeEmptyForMode' },
  { pattern: /^SSH connection '(.+)' not found$/, key: 'validation.sshConnectionNotFound' },
]

export function translateValidationMessage(message: string): string {
  const directKey = VALIDATION_MAP[message]
  if (directKey) return i18n.t(directKey)

  for (const { pattern, key } of VALIDATION_PATTERNS) {
    const match = message.match(pattern)
    if (match) {
      return i18n.t(key, { mode: match[1], id: match[1] })
    }
  }

  return message
}

export function translateValidationErrors(
  details: { field: string; message: string }[],
): { field: string; message: string }[] {
  return details.map((d) => ({
    field: d.field,
    message: translateValidationMessage(d.message),
  }))
}
