import type { Page } from '@playwright/test';

export async function loginAsOperador(page: Page) {
	// Mock WhatsApp stream to avoid QR code overlay blocking UI
	await page.route('**/api/whatsapp/stream', (route) => {
		route.fulfill({
			status: 200,
			contentType: 'text/event-stream',
			headers: { 'Cache-Control': 'no-cache', 'Connection': 'keep-alive' },
			body: 'data: {"connected":true,"qr":""}\n\n',
		});
	});
	// Auth is pre-loaded via storageState from auth.setup.ts
	await page.goto('/operador');
	// Wait for client list to render (proves page + data are ready)
	await page.locator('[data-testid="client-card"]').first().waitFor();
}

export async function selectFirstClient(page: Page) {
	await page.locator('[data-testid="client-card"]').first().click();
	// Wait for the input bar to appear (proves detail view loaded)
	await page.locator('input[placeholder="Escreve aqui..."]').waitFor();
}

export async function selectClientByName(page: Page, name: string) {
	await page.locator('[data-testid="client-card"]').filter({ hasText: name }).click();
	await page.locator('input[placeholder="Escreve aqui..."]').waitFor();
}

export async function switchToGenerateMode(page: Page) {
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	// Wait for the generate-mode placeholder to appear
	await page.locator('input[placeholder="Sobre o que é o post?"]').waitFor();
}

export async function openNewClientForm(page: Page) {
	await page.getByRole('button', { name: '+ Novo Cliente' }).click();
	await page.locator('button[aria-label="Gravar descrição"]').waitFor();
}

export async function openInfoScreen(page: Page) {
	await page.locator('button.min-w-0.flex-1').first().click();
	await page.locator('button').filter({ hasText: 'Voltar' }).first().waitFor();
}
