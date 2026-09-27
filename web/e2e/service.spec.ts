import { test, expect } from '@playwright/test';
const email = process.env.NEXTROLE_TEST_ADMIN || 'admin@nextrole.local';
const password = process.env.NEXTROLE_TEST_PASSWORD;
test.skip(!password, 'NEXTROLE_TEST_PASSWORD is required for browser integration checks');

test('한국어 로그인, 프로필 저장, What-if, 저장, 새로고침과 전체 페이지', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => {
    const url = new URL(response.url());
    if (url.pathname.startsWith('/api/v1/') && response.status() >= 400 && !(url.pathname === '/api/v1/me' && response.status() === 401)) errors.push(`${response.status()} ${url.pathname}`);
  });
  await page.goto('/login');
  await expect(page.getByText(/v1\.0\.0/).first()).toBeVisible();
  await page.getByLabel(/이메일/).fill(email);
  await page.getByPlaceholder('비밀번호를 입력하세요').fill(password!);
  await page.getByRole('button', { name: '로그인', exact: true }).click();
  await expect(page.locator('.sidebar')).toBeVisible();
  await page.goto('/profile');
  await page.getByRole('button', { name: '예시 경력 채우기' }).click();
  await page.getByRole('button', { name: '경력 저장', exact: true }).click();
  await expect(page.getByText('경력 프로필을 저장했습니다.')).toBeVisible();
  await page.goto('/simulator?job=ai-platform');
  await expect(page.locator('.score-change')).toBeVisible();
  const before = await page.locator('.score-change').innerText();
  await page.getByRole('checkbox', { name: 'Python', exact: true }).check();
  await expect.poll(() => page.locator('.score-change').innerText()).not.toBe(before);
  await page.getByRole('button', { name: '시뮬레이션 저장', exact: true }).click();
  await expect(page.getByText('시뮬레이션을 저장했습니다.')).toBeVisible();
  for (const path of ['/discover','/compare','/roadmap','/opportunities','/saved','/keys','/api','/preferences','/admin','/admin/users','/admin/general','/admin/ai','/admin/security','/admin/scoring','/admin/providers','/admin/connectors','/admin/audit']) {
    await page.goto(path);
    await expect(page.locator('main h1')).toBeVisible();
    await page.reload();
    await expect(page.locator('main h1')).toBeVisible();
    expect(new URL(page.url()).pathname).toBe(path);
    await expect(page.getByText('페이지를 찾을 수 없습니다')).toHaveCount(0);
  }
  expect(errors).toEqual([]);
});

test('모바일 메뉴와 가로 넘침', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/login');
  await page.getByLabel(/이메일/).fill(email);
  await page.getByPlaceholder('비밀번호를 입력하세요').fill(password!);
  await page.getByRole('button', { name: '로그인', exact: true }).click();
  await expect(page.getByRole('button', { name: '메뉴 열기' })).toBeVisible();
  await page.getByRole('button', { name: '메뉴 열기' }).click();
  await expect(page.locator('.sidebar')).toBeVisible();
  await page.locator('.sidebar').getByRole('link', { name: /내 경력/ }).click();
  await expect(page.locator('main h1')).toBeVisible();
  for (const path of ['/','/simulator?job=ai-platform','/profile','/admin/connectors']) {
    await page.goto(path); await expect(page.locator('main h1')).toBeVisible();
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 2);
    expect(overflow, `horizontal overflow at ${path}`).toBe(false);
  }
});

test('개인 키 발급·회전·폐기와 실제 API 인증', async ({ page }) => {
  await page.request.post('/api/v1/auth/login', { data: { email, password } });
  await page.goto('/keys');
  const name = `브라우저 검증 ${Date.now()}`;
  await page.getByRole('button', { name: 'API 키 만들기', exact: true }).click();
  await page.getByLabel('키 이름').fill(name);
  await page.getByLabel('유효기간 (일)').fill('30');
  await page.getByRole('button', { name: '저장', exact: true }).click();
  const secret = page.locator('pre.secret');
  await expect(secret).toBeVisible();
  const first = (await secret.innerText()).trim();
  expect((await page.request.get('/api/v1/profile', { headers: { Authorization: `Bearer ${first}` } })).status()).toBe(200);
  await page.getByRole('dialog').filter({ has: secret }).locator('.mantine-Modal-close').click();
  const row = page.getByRole('row').filter({ hasText: name });
  await row.getByRole('button', { name: '회전', exact: true }).click();
  await page.getByRole('button', { name: '새 키 발급 및 기존 키 폐기', exact: true }).click();
  await expect(secret).toBeVisible();
  const rotated = (await secret.innerText()).trim();
  expect(rotated).not.toBe(first);
  expect((await page.request.get('/api/v1/profile', { headers: { Authorization: `Bearer ${first}` } })).status()).toBe(401);
  expect((await page.request.get('/api/v1/profile', { headers: { Authorization: `Bearer ${rotated}` } })).status()).toBe(200);
  await page.getByRole('dialog').filter({ has: secret }).locator('.mantine-Modal-close').click();
  await row.getByRole('button', { name: '폐기', exact: true }).click();
  await page.getByRole('button', { name: '키 폐기', exact: true }).click();
  await expect(row.getByText('폐기됨')).toBeVisible();
  expect((await page.request.get('/api/v1/profile', { headers: { Authorization: `Bearer ${rotated}` } })).status()).toBe(401);
});

test('오프라인 AI 스트리밍과 이력서 텍스트 업로드', async ({ page }) => {
  await page.request.post('/api/v1/auth/login', { data: { email, password } });
  await page.goto('/profile');
  await page.locator('input[type=file]').setInputFiles({ name: 'career.txt', mimeType: 'text/plain', buffer: Buffer.from('Java 개발자로 10년 근무했고 Docker와 Kubernetes 운영 경험이 있습니다.', 'utf8') });
  await expect(page.getByLabel('나의 경력 이야기')).toHaveValue(/Java 개발자/);
  await page.getByRole('button', { name: '경력에서 역량 추출', exact: true }).click();
  await expect(page.locator('.skill-editor').filter({ hasText: 'Java' }).first()).toBeVisible();
  await page.goto('/simulator?job=ai-platform');
  await page.getByRole('button', { name: '스트리밍 설명 보기', exact: true }).click();
  await expect(page.getByText('오프라인 규칙 기반 설명', { exact: true })).toBeVisible();
  await expect(page.locator('.ai-output')).toContainText('오프라인 계산 근거');
});
