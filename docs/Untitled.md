\# 任务：编写“两步走”链上服务播种脚本 (Seed Service Script)



我们需要编写一个独立的 TypeScript 脚本（如 `scripts/seed_tarot_service.ts`），用于在链上真实发布一个“塔罗牌占卜”服务。

请利用现有的钱包和 MetaID 广播模块，严格按以下两步顺序执行：



\## Step 1: 上传 REMOTE-SKILL.md 文件

1. 生成一个临时的 markdown 文件,叫 remote-skill1.md，内容如下：

   `# AI Tarot Reader\nThis is a remote skill for Tarot reading. Send a prompt to get your fortune.`

2. 将这个文件，用 createPIN 方法，上链到 MetaID 的 `/file/remote-skill` 路径下。 content-type 为 text/markdown
3. 广播交易并获取该交易的 `TXID`。根据 MetaID 规则，推导或获取该文件的 `PINID`（通常为 TXID + 'i0'）。



\## Step 2: 发布 skill-service-public 协议

1. 拿到上述的 `PINID` 后，构造以下 JSON 对象：

   {

​     "serviceName": "ai-tarot-reader",

​     "displayName": "AI 塔罗牌大师 (1 DOGE 测试版)",

​     "description": "为你解答近期的财运、事业与爱情疑惑。请在订单中输入你的问题，全自动接单并加密回复。",

​     "price": 100000000, // 假设 1 DOGE 的 SATS 数量

​     "currency": "SATS-DOGE", 

​     "remoteSkillPinId": "<这里填入 Step 1 获取到的真实 PINID>",

​     "availableBeforeBTCHeight": 9999999

   }

2. 将该 JSON 广播到 `/protocols/skill-service-public` 路径下。



\## 执行与日志

请在脚本中使用适当的 `await` 和延迟，确保两笔交易顺序成功。

成功后，在控制台高亮打印：

[SUCCESS] 步骤 1: 技能文档上链成功，PINID: xxxxxx

[SUCCESS] 步骤 2: 服务广场协议上架成功，TXID: xxxxxx



请提供完整的脚本代码并帮我执行。