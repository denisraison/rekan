import { test } from '@playwright/test';
import { loginAsOperador, selectFirstClient, switchToGenerateMode } from './helpers';

test.use({ viewport: { width: 320, height: 568 } });

test('screenshot: chat large text', async ({ page }) => {
  await loginAsOperador(page);
  await page.addStyleTag({ content: 'html { zoom: 1.35 !important; }' });
  await selectFirstClient(page);
  await page.screenshot({ path: '/tmp/a11y-chat-large.png' });
});

test('screenshot: generate large text', async ({ page }) => {
  await loginAsOperador(page);
  await page.addStyleTag({ content: 'html { zoom: 1.35 !important; }' });
  await selectFirstClient(page);
  await switchToGenerateMode(page);
  await page.screenshot({ path: '/tmp/a11y-generate-large.png' });
});

test('screenshot: chat XL text', async ({ page }) => {
  await loginAsOperador(page);
  await page.addStyleTag({ content: 'html { zoom: 1.5 !important; }' });
  await selectFirstClient(page);
  await page.screenshot({ path: '/tmp/a11y-chat-xl.png' });
});
