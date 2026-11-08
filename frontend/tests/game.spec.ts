import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";

async function hasInk(page: Page) {
  return page.getByLabel("绘画画板").evaluate((el: HTMLCanvasElement) => {
    const data = el
      .getContext("2d")!
      .getImageData(0, 0, el.width, el.height).data;
    for (let i = 0; i < data.length; i += 4) if (data[i]! < 200) return true;
    return false;
  });
}
async function draw(page: Page) {
  const box = (await page.getByLabel("绘画画板").boundingBox())!;
  await page.mouse.move(box.x + 80, box.y + 90);
  await page.mouse.down();
  await page.mouse.move(box.x + 180, box.y + 160, { steps: 16 });
  await page.mouse.up();
}

test("two players draw, reconnect, guess, finish and inspect history", async ({
  page,
  browser,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("/");
  await page.getByLabel("你的昵称").fill("小画家");
  await page.getByLabel("游戏轮数", { exact: true }).selectOption("1");
  await page.getByLabel("每次绘画", { exact: true }).selectOption("30");
  await page.getByRole("button", { name: "创建我的快乐房间" }).click();
  await expect(
    page.getByRole("heading", { name: /^灵感集合中/ }),
  ).toBeVisible();
  await page.screenshot({
    path: "test-results/room-lobby.png",
    fullPage: true,
  });
  const context = await browser.newContext({
      viewport: { width: 1440, height: 1000 },
    }),
    guest = await context.newPage();
  guest.on("pageerror", (error) => errors.push(error.message));
  try {
    await guest.goto(page.url());
    await guest.getByLabel("你的昵称").fill("猜词高手");
    await guest.getByRole("button", { name: "加入朋友的房间" }).click();
    await expect(
      page.getByRole("button", { name: "开始游戏", exact: true }),
    ).toBeEnabled();
    await page.getByRole("button", { name: "开始游戏", exact: true }).click();
    await expect(page.getByText("这次，画点什么？")).toBeVisible();
    await expect(guest.locator(".word-options")).toHaveCount(0);
    await page.locator(".word-options button").first().click();
    const word = await page.locator(".secret-word").innerText();
    await expect(guest.locator(".secret-word")).toHaveCount(0);
    await draw(page);
    await expect.poll(() => hasInk(guest)).toBe(true);
    await page.getByRole("button", { name: "撤销", exact: true }).click();
    await expect.poll(() => hasInk(guest)).toBe(false);
    await draw(page);
    await expect.poll(() => hasInk(guest)).toBe(true);
    await guest.reload();
    await expect(guest.locator(".word-blanks")).toBeVisible();
    await expect.poll(() => hasInk(guest)).toBe(true);
    await expect(guest.locator(".secret-word")).toHaveCount(0);
    await page.screenshot({
      path: "test-results/room-drawing.png",
      fullPage: true,
    });
    await guest.getByLabel("猜词或聊天").fill(word);
    await guest.getByLabel("发送", { exact: true }).click();
    await expect(page.locator(".answer-banner strong")).toHaveText(word);
    await expect(guest.getByText("这次，画点什么？")).toBeVisible({
      timeout: 10000,
    });
    await guest.locator(".word-options button").first().click();
    const second = await guest.locator(".secret-word").innerText();
    await page.getByLabel("猜词或聊天").fill(second);
    await page.getByLabel("发送", { exact: true }).click();
    await expect(page.getByText("本场最佳脑洞", { exact: true })).toBeVisible({
      timeout: 10000,
    });
    await page.getByRole("button", { name: "我的战绩" }).click();
    await expect(page.locator(".history-list article")).toHaveCount(1);
    await expect(page.locator(".history-list")).toContainText("小画家");
    expect(errors).toEqual([]);
  } finally {
    await context.close();
  }
});

test("mobile layout and touch drawing work without horizontal overflow", async ({
  browser,
}) => {
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    isMobile: true,
    hasTouch: true,
  });
  const page = await context.newPage(),
    helperContext = await browser.newContext(),
    helper = await helperContext.newPage();
  try {
    await page.goto("/");
    await page.screenshot({
      path: "test-results/home-mobile.png",
      fullPage: true,
    });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.getByLabel("你的昵称").fill("手机画手");
    await page.getByRole("button", { name: "创建我的快乐房间" }).click();
    await expect(
      page.getByRole("heading", { name: /^灵感集合中/ }),
    ).toBeVisible();
    await helper.goto(page.url());
    await helper.getByLabel("你的昵称").fill("朋友");
    await helper.getByRole("button", { name: "加入朋友的房间" }).click();
    await page.getByRole("button", { name: "开始游戏", exact: true }).click();
    await page.locator(".word-options button").first().click();
    await expect(page.locator(".secret-word")).toBeVisible();
    await expect(helper.locator(".word-blanks")).toBeVisible();
    const box = (await page.getByLabel("绘画画板").boundingBox())!;
    const touch = await context.newCDPSession(page);
    await touch.send("Input.dispatchTouchEvent", {
      type: "touchStart",
      touchPoints: [{ x: box.x + 40, y: box.y + 50 }],
    });
    await touch.send("Input.dispatchTouchEvent", {
      type: "touchMove",
      touchPoints: [{ x: box.x + 140, y: box.y + 100 }],
    });
    await touch.send("Input.dispatchTouchEvent", {
      type: "touchEnd",
      touchPoints: [],
    });
    await expect.poll(() => hasInk(helper)).toBe(true);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: "test-results/room-mobile.png",
      fullPage: true,
    });
  } finally {
    await context.close();
    await helperContext.close();
  }
});

test("home desktop visual check", async ({ page }) => {
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: /你的灵魂画作/ }),
  ).toBeVisible();
  await page.screenshot({
    path: "test-results/home-desktop.png",
    fullPage: true,
  });
});

test("invalid room returns immediately and a new tab takes over the same identity", async ({
  page,
  context,
}) => {
  await page.goto("/?room=AAAAAAAA");
  await page.getByLabel("你的昵称").fill("同一玩家");
  await page.getByRole("button", { name: "加入朋友的房间" }).click();
  await expect(page.getByRole("status")).toHaveText(/房间不存在或已过期/);
  await expect(
    page.getByRole("heading", { name: /你的灵魂画作/ }),
  ).toBeVisible();
  await page.getByRole("tab", { name: "创建房间" }).click();
  await page.getByRole("button", { name: "创建我的快乐房间" }).click();
  await expect(
    page.getByRole("heading", { name: /^灵感集合中/ }),
  ).toBeVisible();
  const second = await context.newPage();
  await second.goto(page.url());
  await expect(
    second.getByRole("heading", { name: /^灵感集合中/ }),
  ).toBeVisible();
  await expect(page.getByRole("status")).toHaveText(/已在另一个页面/);
  await expect(
    page.getByRole("heading", { name: /你的灵魂画作/ }),
  ).toBeVisible();
  await second.close();
});
