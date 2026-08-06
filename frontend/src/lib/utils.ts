// Minimal class-name joiner (plain-CSS replacement for clsx/twMerge).
type ClassValue = string | number | boolean | null | undefined;

export function cn(...classes: ClassValue[]): string {
    return classes.filter(Boolean).join(' ');
}
