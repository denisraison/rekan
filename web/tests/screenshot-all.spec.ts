import { test, type Page } from '@playwright/test';
import { loginAsOperador, selectFirstClient, switchToGenerateMode, openNewClientForm, openInfoScreen } from './helpers';

const OUT = 'test-results/screenshots';

async function captureAllViews(page: Page, prefix: string) {
	await loginAsOperador(page);

	// 1. Client list
	await page.screenshot({ path: `${OUT}/${prefix}01-client-list.png` });

	// 2. Chat (empty state)
	await selectFirstClient(page);
	await page.screenshot({ path: `${OUT}/${prefix}02-chat-empty.png` });

	// 3. Generate mode
	await switchToGenerateMode(page);
	await page.screenshot({ path: `${OUT}/${prefix}03-generate-mode.png` });

	// 4. Info screen
	await openInfoScreen(page);
	await page.screenshot({ path: `${OUT}/${prefix}04-info-screen.png` });

	// 5. Back to list, then new client form
	await page.goBack();
	await page.locator('[data-testid="input-bar"]').waitFor();
	await page.goBack();
	await page.getByRole('button', { name: '+ Novo' }).waitFor();
	await openNewClientForm(page);
	await page.screenshot({ path: `${OUT}/${prefix}05-new-client.png` });
}

// --- Default: Moto G (360x740) ---
test.describe('Moto G (360x740)', () => {
	test.use({ viewport: { width: 360, height: 740 } });
	test('capture all operator views', async ({ page }) => {
		await captureAllViews(page, '');
	});
});

// --- Worst case: small display + large font ---
// Android "Display size: Large" on Moto G shrinks logical viewport to ~300px.
// Combined with "Font size: Large" (1.3x, root 21px).
test.describe('Small display + large font (300x600 @ 21px)', () => {
	test.use({ viewport: { width: 300, height: 600 } });
	test('capture all operator views', async ({ page }) => {
		await page.addStyleTag({ content: 'html { font-size: 21px !important; }' });
		await captureAllViews(page, 'lg-');
	});
});
