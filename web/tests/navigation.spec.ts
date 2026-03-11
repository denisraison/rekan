import { expect, test } from '@playwright/test';
import { loginAsOperador, selectFirstClient } from './helpers';

// Use mobile viewport so mobileView transitions are active
test.use({ viewport: { width: 375, height: 667 } });

test.describe('Browser back button navigation (Wave 5)', () => {
	test('selecting a client pushes history, back returns to list', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);

		// Detail view is showing (input bar visible)
		await expect(page.locator('input[placeholder="Escreve aqui..."]')).toBeVisible();

		// Press browser back
		await page.goBack();

		// Should return to list view (client buttons visible, input bar gone)
		await page.locator('button.text-left.border-b').first().waitFor();
		await expect(page.locator('input[placeholder="Escreve aqui..."]')).not.toBeVisible();
	});

	test('opening info screen pushes history, back returns to detail', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);

		// Tap the client name to open info screen
		const nameBtn = page.locator('button.min-w-0.flex-1').first();
		await nameBtn.click();

		// Info screen should be visible (has "Voltar" and "Editar" buttons)
		const infoVoltar = page.locator('button').filter({ hasText: 'Voltar' }).first();
		await expect(infoVoltar).toBeVisible();

		// Press browser back
		await page.goBack();

		// Should return to detail view (input bar visible)
		await expect(page.locator('input[placeholder="Escreve aqui..."]')).toBeVisible();
	});

	test('back from detail clears selectedId', async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);

		// Press browser back
		await page.goBack();

		// List view: no detail header with Voltar button visible
		await page.locator('button.text-left.border-b').first().waitFor();

		// Select a different client to confirm selectedId was cleared
		const clients = page.locator('button.text-left.border-b');
		const count = await clients.count();
		if (count > 1) {
			await clients.nth(1).click();
		} else {
			await clients.first().click();
		}
		await expect(page.locator('input[placeholder="Escreve aqui..."]')).toBeVisible();
	});
});
