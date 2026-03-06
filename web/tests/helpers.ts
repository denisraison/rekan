import type { Page } from '@playwright/test';

export async function loginAsOperador(page: Page) {
	// Auth is pre-loaded via storageState from auth.setup.ts
	await page.goto('/operador');
	// Wait for client list to render (proves page + data are ready)
	await page.locator('button.text-left.border-b').first().waitFor();
}

export async function selectFirstClient(page: Page) {
	await page.locator('button.text-left.border-b').first().click();
	// Wait for the input bar to appear (proves detail view loaded)
	await page.locator('input[placeholder="Mensagem..."]').waitFor();
}

export async function switchToGenerateMode(page: Page) {
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	// Wait for the generate-mode placeholder to appear
	await page.locator('input[placeholder="Descreva o post..."]').waitFor();
}
