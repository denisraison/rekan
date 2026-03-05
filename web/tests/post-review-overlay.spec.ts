import { expect, test } from '@playwright/test';

test.use({ ignoreHTTPSErrors: true, baseURL: 'https://localhost:5173' });

async function loginAsOperador(page: any) {
	await page.goto('/entrar');
	await page.getByLabel('Email').fill('operador@rekan.local');
	await page.getByLabel('Senha').fill('senha1234567');
	await page.getByRole('button', { name: 'Entrar' }).click();
	await page.waitForURL('**/operador**');
	await page.waitForTimeout(2000);
}

async function selectFirstClient(page: any) {
	// Client list buttons have border-b and text-left classes
	const clientButton = page.locator('button.text-left.border-b').first();
	await clientButton.click();
	await page.waitForTimeout(1000);
}

async function switchToGenerateMode(page: any) {
	// The mode toggle pill says "Post" (coral) when in chat mode
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await page.waitForTimeout(300);
}

async function generatePost(page: any, prompt: string) {
	const input = page.locator('input[placeholder="Descreva o post..."]');
	await input.fill(prompt);
	await page.getByRole('button', { name: 'Gerar' }).click();
	// Wait for overlay
	await page.getByText('Post gerado').waitFor({ timeout: 30000 });
}

test.describe('Post review overlay (Wave 3)', () => {
	test.beforeEach(async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
		await switchToGenerateMode(page);
	});

	test('generate shows full-screen overlay with editable caption', async ({ page }) => {
		await generatePost(page, 'Post sobre corte de cabelo');

		// Verify editable caption textarea
		const captionTextarea = page.locator('textarea');
		await expect(captionTextarea).toBeVisible();
		const captionValue = await captionTextarea.inputValue();
		expect(captionValue.length).toBeGreaterThan(0);

		// Verify sections
		await expect(page.getByText('Legenda')).toBeVisible();
		await expect(page.getByText('Hashtags')).toBeVisible();

		// Verify action buttons
		await expect(page.getByRole('button', { name: 'Enviar pelo WhatsApp' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Descartar' })).toBeVisible();

		await page.screenshot({ path: '/tmp/overlay-open.png', fullPage: true });
	});

	test('Voltar closes overlay but preserves result, re-openable', async ({ page }) => {
		await generatePost(page, 'Post sobre salao');

		// The overlay header has "Voltar" and the detail header may also have it.
		// The overlay's Voltar is inside the overlay (absolute inset-0 z-10).
		const overlayVoltar = page.locator('.absolute.inset-0.z-10 button', { hasText: 'Voltar' });
		await overlayVoltar.click();

		// Overlay should be hidden (check the overlay header span, not the chip button)
		const overlayTitle = page.locator('.absolute.inset-0.z-10');
		await expect(overlayTitle).not.toBeVisible();

		// "Ver post gerado" chip should be visible
		const reopenBtn = page.getByRole('button', { name: 'Ver post gerado' });
		await expect(reopenBtn).toBeVisible();

		// Re-open
		await reopenBtn.click();
		await expect(overlayTitle).toBeVisible();
	});

	test('Descartar clears everything', async ({ page }) => {
		await generatePost(page, 'Post sobre beleza');

		await page.getByRole('button', { name: 'Descartar' }).click();

		await expect(page.getByText('Post gerado')).not.toBeVisible();
		await expect(page.getByRole('button', { name: 'Ver post gerado' })).not.toBeVisible();
	});

	test('editing caption persists through Voltar round-trip', async ({ page }) => {
		await generatePost(page, 'Post sobre corte');

		// Edit the caption
		const captionTextarea = page.locator('textarea');
		await captionTextarea.clear();
		await captionTextarea.fill('Legenda editada');

		// Close and reopen
		const overlayVoltar = page.locator('.absolute.inset-0.z-10 button', { hasText: 'Voltar' });
		await overlayVoltar.click();
		await page.getByRole('button', { name: 'Ver post gerado' }).click();

		// Caption should still have the edited value
		await expect(captionTextarea).toHaveValue('Legenda editada');
	});
});
