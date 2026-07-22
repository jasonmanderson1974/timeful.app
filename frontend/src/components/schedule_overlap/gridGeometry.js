import { clamp } from "@/utils"

/**
 * Pure grid-geometry math for ScheduleOverlap's drag-select grid.
 *
 * These were inline in ScheduleOverlap.vue's methods (this-dependent); extracted
 * here as pure functions of their inputs so they can be unit-tested (TODO B3).
 * dragGridMixin's clampRow/clampCol/getRowColFromXY delegate to these, passing
 * the relevant component state as args — behavior is unchanged.
 */

/**
 * Clamp a row index to the valid range for the current view.
 * @param {number} row
 * @param {{daysOnly:boolean, monthDaysLength:number, timesLength:number}} opts
 */
export function clampRow(row, { daysOnly, monthDaysLength, timesLength }) {
  if (daysOnly) {
    return clamp(row, 0, Math.floor(monthDaysLength / 7))
  }
  return clamp(row, 0, timesLength - 1)
}

/**
 * Clamp a column index to the valid range for the current view.
 * @param {number} col
 * @param {{daysOnly:boolean, daysLength:number}} opts
 */
export function clampCol(col, { daysOnly, daysLength }) {
  if (daysOnly) {
    return clamp(col, 0, 7 - 1)
  }
  return clamp(col, 0, daysLength - 1)
}

/**
 * Given an x/y pixel position (relative to the grid element), return the
 * {row, col} of the timeslot under it.
 *
 * @param {number} x
 * @param {number} y
 * @param {object} opts
 * @param {boolean} opts.daysOnly
 * @param {number} opts.timeslotWidth
 * @param {number} opts.timeslotHeight
 * @param {number[]} opts.columnOffsets   cumulative left offsets per column (non-daysOnly)
 * @param {number} opts.firstSplitLength  number of rows before the split gap
 * @param {number} opts.splitGapHeight    pixel height of the split gap
 * @param {number} opts.monthDaysLength
 * @param {number} opts.timesLength
 * @param {number} opts.daysLength
 */
export function getRowColFromXY(
  x,
  y,
  {
    daysOnly,
    timeslotWidth,
    timeslotHeight,
    columnOffsets,
    firstSplitLength,
    splitGapHeight,
    monthDaysLength,
    timesLength,
    daysLength,
  }
) {
  let col = Math.floor(x / timeslotWidth)
  if (!daysOnly) {
    col = columnOffsets.length
    for (let i = 0; i < columnOffsets.length; ++i) {
      if (x < columnOffsets[i]) {
        col = i - 1
        break
      }
    }
  }
  let row = Math.floor(y / timeslotHeight)

  // Account for split gap
  if (!daysOnly && row > firstSplitLength) {
    const adjustedRow = Math.floor((y - splitGapHeight) / timeslotHeight)
    if (adjustedRow >= firstSplitLength) {
      // Make sure we don't go to a lesser index
      row = adjustedRow
    }
  }

  row = clampRow(row, { daysOnly, monthDaysLength, timesLength })
  col = clampCol(col, { daysOnly, daysLength })
  return { row, col }
}
