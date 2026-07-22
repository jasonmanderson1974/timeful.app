import { describe, it, expect } from "vitest"
import { clampRow, clampCol, getRowColFromXY } from "./gridGeometry"

describe("clampRow", () => {
  it("clamps to floor(monthDaysLength / 7) when daysOnly", () => {
    const opts = { daysOnly: true, monthDaysLength: 35, timesLength: 48 }
    expect(clampRow(3, opts)).toBe(3)
    expect(clampRow(99, opts)).toBe(5) // floor(35/7) = 5
    expect(clampRow(-2, opts)).toBe(0)
  })

  it("clamps to timesLength - 1 when not daysOnly", () => {
    const opts = { daysOnly: false, monthDaysLength: 35, timesLength: 48 }
    expect(clampRow(10, opts)).toBe(10)
    expect(clampRow(100, opts)).toBe(47)
    expect(clampRow(-5, opts)).toBe(0)
  })
})

describe("clampCol", () => {
  it("clamps to 0..6 (a week) when daysOnly", () => {
    const opts = { daysOnly: true, daysLength: 3 }
    expect(clampCol(4, opts)).toBe(4)
    expect(clampCol(9, opts)).toBe(6)
    expect(clampCol(-1, opts)).toBe(0)
  })

  it("clamps to daysLength - 1 when not daysOnly", () => {
    const opts = { daysOnly: false, daysLength: 5 }
    expect(clampCol(2, opts)).toBe(2)
    expect(clampCol(10, opts)).toBe(4)
    expect(clampCol(-3, opts)).toBe(0)
  })
})

describe("getRowColFromXY", () => {
  // In daysOnly (month) view, col is a simple x/width division, clamped to a week.
  it("computes row/col by simple division in daysOnly view", () => {
    const got = getRowColFromXY(45, 25, {
      daysOnly: true,
      timeslotWidth: 20,
      timeslotHeight: 10,
      columnOffsets: [],
      firstSplitLength: 0,
      splitGapHeight: 0,
      monthDaysLength: 35,
      timesLength: 48,
      daysLength: 7,
    })
    expect(got).toEqual({ row: 2, col: 2 }) // floor(45/20)=2, floor(25/10)=2
  })

  // In the time grid, col is derived from cumulative columnOffsets: the column
  // is the last offset the x-position is past.
  it("derives col from columnOffsets when not daysOnly", () => {
    const opts = {
      daysOnly: false,
      timeslotWidth: 50,
      timeslotHeight: 10,
      columnOffsets: [0, 50, 100, 150, 200],
      firstSplitLength: 100,
      splitGapHeight: 20,
      monthDaysLength: 35,
      timesLength: 48,
      daysLength: 4,
    }
    // x=120 -> first offset greater than 120 is index 3 (150) -> col = 3 - 1 = 2
    expect(getRowColFromXY(120, 35, opts)).toEqual({ row: 3, col: 2 })
  })

  it("clamps col to daysLength-1 when x is past the last offset", () => {
    const opts = {
      daysOnly: false,
      timeslotWidth: 50,
      timeslotHeight: 10,
      columnOffsets: [0, 50, 100],
      firstSplitLength: 100,
      splitGapHeight: 20,
      monthDaysLength: 35,
      timesLength: 48,
      daysLength: 2,
    }
    // x=999 past all offsets -> col = columnOffsets.length (3) -> clamped to daysLength-1 = 1
    expect(getRowColFromXY(999, 5, opts).col).toBe(1)
  })

  // When the y position is below the split gap, the gap height is subtracted so
  // the row maps back onto the second block of times.
  it("accounts for the split gap below the first block", () => {
    const opts = {
      daysOnly: false,
      timeslotWidth: 50,
      timeslotHeight: 10,
      columnOffsets: [0, 50, 100],
      firstSplitLength: 2,
      splitGapHeight: 20,
      monthDaysLength: 35,
      timesLength: 48,
      daysLength: 2,
    }
    // y=55 -> raw row = 5 (> firstSplitLength 2); adjusted = floor((55-20)/10)=3
    // 3 >= 2 so row becomes 3 (not 5).
    expect(getRowColFromXY(60, 55, opts).row).toBe(3)
  })

  it("does not adjust for the gap when row is within the first block", () => {
    const opts = {
      daysOnly: false,
      timeslotWidth: 50,
      timeslotHeight: 10,
      columnOffsets: [0, 50, 100],
      firstSplitLength: 4,
      splitGapHeight: 20,
      monthDaysLength: 35,
      timesLength: 48,
      daysLength: 2,
    }
    // y=25 -> row = 2, which is <= firstSplitLength(4), so no gap adjustment.
    expect(getRowColFromXY(60, 25, opts).row).toBe(2)
  })
})
