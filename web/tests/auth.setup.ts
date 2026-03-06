import { test as setup } from '@playwright/test';

const STORAGE_STATE = 'tests/.auth/operador.json';

setup('authenticate as operador', async ({ page }) => {
	// Mock WhatsApp stream to avoid QR code overlay blocking UI
	await page.route('**/api/whatsapp/stream', (route) => {
		route.fulfill({
			status: 200,
			contentType: 'text/event-stream',
			headers: { 'Cache-Control': 'no-cache', 'Connection': 'keep-alive' },
			body: 'data: {"connected":true,"qr":""}\n\n',
		});
	});
	await page.goto('/entrar');
	await page.getByLabel('Email').fill('operador@rekan.local');
	await page.getByLabel('Senha').fill('senha1234567');
	await page.getByRole('button', { name: 'Entrar' }).click();
	await page.locator('button.text-left.border-b').first().waitFor();
	await page.context().storageState({ path: STORAGE_STATE });
});
