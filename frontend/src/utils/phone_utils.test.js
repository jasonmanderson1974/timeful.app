import { describe, test, expect } from "vitest"
import { formatPhone } from "@/utils/phone_utils"

describe("formatPhone", () => {
  test("formats a US 10-digit number", () => {
    expect(formatPhone("1234567890")).toBe("(123) 456-7890")
  })

  test("strips a leading US 1 (11 digits)", () => {
    expect(formatPhone("11234567890")).toBe("(123) 456-7890")
    expect(formatPhone("+1 123 456 7890")).toBe("(123) 456-7890")
  })

  test("ignores existing separators", () => {
    expect(formatPhone("123-456-7890")).toBe("(123) 456-7890")
    expect(formatPhone("(123) 456-7890")).toBe("(123) 456-7890") // idempotent
  })

  test("leaves non-US / partial numbers as typed (trimmed)", () => {
    expect(formatPhone("+44 20 7946 0958")).toBe("+44 20 7946 0958")
    expect(formatPhone("12345")).toBe("12345")
    expect(formatPhone("  555-12  ")).toBe("555-12")
  })

  test("passes through empty/nullish", () => {
    expect(formatPhone("")).toBe("")
    expect(formatPhone(null)).toBe(null)
    expect(formatPhone(undefined)).toBe(undefined)
  })
})
