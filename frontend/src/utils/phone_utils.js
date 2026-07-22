// Beautify a US 10-digit phone number (or 11 with a leading 1) into
// "(123) 456-7890". Anything else (partial, international, empty) is returned
// trimmed but otherwise as-is, so freeform input is preserved.
export const formatPhone = (value) => {
  if (!value) return value
  const trimmed = String(value).trim()
  const digits = trimmed.replace(/\D/g, "")
  let ten = null
  if (digits.length === 10) {
    ten = digits
  } else if (digits.length === 11 && digits.charAt(0) === "1") {
    ten = digits.slice(1)
  }
  if (ten) {
    return `(${ten.slice(0, 3)}) ${ten.slice(3, 6)}-${ten.slice(6)}`
  }
  return trimmed
}
