import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export type WithElementRef<T, U extends HTMLElement = HTMLElement> = T & {
	ref?: U | null;
};

export type WithoutChildren<T> = Omit<T, 'children'> & { children?: never };

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

export async function copyText(text: string) {
	if (navigator.clipboard?.writeText) await navigator.clipboard.writeText(text);
}
