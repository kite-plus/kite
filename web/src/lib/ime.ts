import type { KeyboardEvent } from "react";

/**
 * composing reports a key that belongs to an input method still composing
 * text, such as the Enter that picks a candidate. Safari reports the Enter
 * that ends a composition only by its key code.
 */
export function composing(event: KeyboardEvent): boolean {
  return event.nativeEvent.isComposing || event.keyCode === 229;
}
