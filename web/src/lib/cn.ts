import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * cn merges class names, letting a later class win over an earlier one of the
 * same kind. It follows the shadcn/ui convention so that generated components
 * can be dropped in later without rework.
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
