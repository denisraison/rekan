import { PUBLIC_WHATSAPP_NUMBER } from '$env/static/public';

export function waLink(text: string): string {
	return `https://wa.me/${PUBLIC_WHATSAPP_NUMBER}?text=${encodeURIComponent(text)}`;
}
