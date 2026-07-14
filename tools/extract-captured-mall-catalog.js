const childProcess = require('child_process');
const fs = require('fs');
const path = require('path');

const SOURCE_YUBI_CURRENCY = '玉币';

const CATEGORIES = [
  { categoryId: '1', name: '推荐商品' },
  { categoryId: '2', name: '全部' },
  {
    categoryId: '3',
    name: '消耗品',
    childs: [
      { categoryId: '4', name: '常用' },
      { categoryId: '5', name: '洗点药' },
      { categoryId: '6', name: '符文' },
      { categoryId: '7', name: '宠物药剂' },
      { categoryId: '8', name: '合成' },
    ],
  },
  { categoryId: '9', name: '经验卡' },
  { categoryId: '10', name: '仙果' },
  { categoryId: '11', name: '坐骑' },
  { categoryId: '12', name: '装备' },
  { categoryId: '13', name: '宝石' },
  { categoryId: '14', name: '通行证' },
  {
    categoryId: '15',
    name: '时装',
    childs: [
      { categoryId: '16', name: '永久时装' },
      { categoryId: '17', name: '附加属性' },
    ],
  },
];

function fail(message) {
  process.stderr.write(`${message}\n`);
  process.exit(1);
}

function parseArgs(args) {
  const roots = [];
  let out = '';
  for (let index = 0; index < args.length; index += 1) {
    const value = args[index];
    if (value === '--root') {
      roots.push(args[index + 1] || '');
      index += 1;
      continue;
    }
    if (value === '--out') {
      out = args[index + 1] || '';
      index += 1;
      continue;
    }
    fail(`未知参数: ${value}`);
  }
  if (!out || roots.length === 0 || roots.some((root) => !root)) {
    fail('用法: node tools/extract-captured-mall-catalog.js --out <catalog.json> --root <capture-root> [--root <capture-root>]');
  }
  return { roots, out };
}

function commandKey(text) {
  if (text.length < 4) {
    return null;
  }
  return (text.charCodeAt(0) + text.charCodeAt(2) - 64) * 446 + text.charCodeAt(1) + text.charCodeAt(3) - 64;
}

function readPackets(filePath) {
  return fs.readFileSync(filePath).toString('utf8').split('\0').filter(Boolean);
}

function parseSearchPages(clientPath) {
  return readPackets(clientPath)
    .filter((packet) => commandKey(packet) === 273)
    .map((packet) => packet.slice(4).split(',').map((value) => value.replaceAll('&0;', ',')))
    .filter((values) => Number(values[5]) > 0)
    .map((values) => ({
      categoryId: values[0],
      offset: Number(values[4]),
      limit: Number(values[5]),
    }));
}

function parseProductResponses(serverPath) {
  return readPackets(serverPath)
    .filter((packet) => commandKey(packet) === 50011)
    .map((packet) => packet.slice(4).replaceAll('&0;', ','))
    .map((message) => message.match(/^商品检索详细json:(\[[\s\S]*?\])(?=,50$)/))
    .filter(Boolean)
    .map((match) => JSON.parse(match[1]));
}

function listServerFiles(roots) {
  const command = ['-l', '-a', '商品检索详细json', ...roots, '-g', 'server-to-client*.bin'];
  const output = childProcess.execFileSync('rg', command, { encoding: 'utf8' }).trim();
  return output ? output.split(/\r?\n/).sort() : [];
}

function normalizeItem(item) {
  return {
    name: String(item.name || ''),
    display: String(item.display || ''),
    // &104 is a per-capture expiry timestamp. A catalog row must not freeze a
    // historical timestamp into later purchases; delivery expiry is not yet a
    // confirmed local protocol field.
    description: String(item.description || '').replace(/&104@\d+/g, '&104@0'),
    count: Number(item.count || 1),
  };
}

function buildProduct(row, categoryId, categoryOrder) {
  const items = Array.isArray(row.items) ? row.items.map(normalizeItem) : [];
  return {
    productId: String(row.id),
    categoryId,
    categoryIds: [categoryId],
    categoryOrder: { [categoryId]: categoryOrder },
    name: String(row.name || ''),
    icon: String(row.display || items[0]?.display || ''),
    price: Number(row.price || 0),
    couponPrice: Number(row.dq_price || 0),
    discount: Number(row.discount || 1),
    recommended: categoryId === '1',
    currency: SOURCE_YUBI_CURRENCY,
    description: String(row.description || items[0]?.description || ''),
    items,
  };
}

function sameProductData(left, right) {
  return left.name === right.name
    && left.icon === right.icon
    && left.price === right.price
    && left.couponPrice === right.couponPrice
    && left.discount === right.discount
    && left.description === right.description
    && JSON.stringify(left.items) === JSON.stringify(right.items);
}

function choosePrimaryCategory(categoryIds) {
  if (categoryIds.includes('1')) {
    return '1';
  }
  const leaf = categoryIds.find((categoryId) => !['0', '2', '3', '15'].includes(categoryId));
  return leaf || categoryIds.find((categoryId) => categoryId !== '0') || '2';
}

function categoryRank(categoryId) {
  return Number(categoryId);
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const serverFiles = listServerFiles(options.roots);
  if (serverFiles.length === 0) {
    fail('未找到含 商品检索详细json 的 server-to-client 抓包文件。');
  }

  const products = new Map();
  let responseCount = 0;
  for (const serverPath of serverFiles) {
    const clientPath = path.join(
      path.dirname(serverPath),
      path.basename(serverPath).replace('server-to-client', 'client-to-server'),
    );
    if (!fs.existsSync(clientPath)) {
      fail(`缺少配对客户端抓包: ${clientPath}`);
    }
    const pages = parseSearchPages(clientPath);
    const responses = parseProductResponses(serverPath);
    if (pages.length !== responses.length) {
      fail(`SearchMarket 请求与商品详情响应数量不一致: ${serverPath}; requests=${pages.length}; responses=${responses.length}`);
    }
    responseCount += responses.length;
    for (let index = 0; index < responses.length; index += 1) {
      const page = pages[index];
      const rows = responses[index];
      for (let itemIndex = 0; itemIndex < rows.length; itemIndex += 1) {
        const product = buildProduct(rows[itemIndex], page.categoryId, page.offset + itemIndex);
        const existing = products.get(product.productId);
        if (!existing) {
          products.set(product.productId, product);
          continue;
        }
        if (!sameProductData(existing, product)) {
          fail(`同一商品抓包元数据冲突: productId=${product.productId}`);
        }
        if (!existing.categoryIds.includes(page.categoryId)) {
          existing.categoryIds.push(page.categoryId);
        }
        const currentOrder = existing.categoryOrder[page.categoryId];
        existing.categoryOrder[page.categoryId] = currentOrder === undefined
          ? product.categoryOrder[page.categoryId]
          : Math.min(currentOrder, product.categoryOrder[page.categoryId]);
        existing.recommended = existing.recommended || page.categoryId === '1';
      }
    }
  }

  const catalogProducts = [...products.values()]
    .map((product) => {
      product.categoryIds.sort((left, right) => categoryRank(left) - categoryRank(right));
      product.categoryId = choosePrimaryCategory(product.categoryIds);
      return product;
    })
    .sort((left, right) => Number(left.productId) - Number(right.productId));
  const catalog = {
    schemaVersion: 1,
    capture: {
      requestCommandKey: 273,
      responseCommandKey: 50011,
      responseMessage: '商品检索详细json',
      sourceFileCount: serverFiles.length,
      responseCount,
      uniqueProductCount: catalogProducts.length,
    },
    categories: CATEGORIES,
    products: catalogProducts,
  };
  fs.writeFileSync(options.out, `${JSON.stringify(catalog, null, 2)}\n`, 'utf8');
  process.stdout.write(`generated ${options.out}: ${catalogProducts.length} products from ${responseCount} detail responses\n`);
}

main();
