package main

// These rows are from the captured map191 c_ItemEqualInfo payloads. Equipment
// descriptions resolve through the captured item table in sourceItemShopEquipmentDescription.
const sourceWuliangMap2HealerShopRows = `
0|馒头|own|0.png|f_i_馒头&24@消耗品&25@99&7@200&20@又白又香的馒头&0;饥饿的时候用来充饥.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|10
1|花卷|own|213.png|f_i_花卷&24@消耗品&25@99&7@500&20@又白又香的花卷&0;饥饿的时候用来充饥.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|50
2|包子|own|212.png|f_i_包子&24@消耗品&25@99&7@600&20@带馅的包子&0;看起来非常可口&0;食用后可恢复些气力.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|120
3|小包还元散|own|695.png|f_i_小包还元散&24@消耗品&25@99&7@1500&20@恢复气力的药散.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|350
4|甘露|own|214.png|f_i_甘露&24@消耗品&25@99&8@50&20@恢复精力的甘露.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|10
5|小瓶甘露|own|696.png|f_i_小瓶甘露&24@消耗品&25@99&8@100&20@恢复精力的甘露.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|25
6|中瓶甘露|own|697.png|f_i_中瓶甘露&24@消耗品&25@99&8@200&20@恢复精力的甘露.&103@0&104@0&105@&107@&108@0|1|2|铜钱|163.png|60
7|大瓶甘露|own|698.png|f_i_大瓶甘露&24@消耗品&25@99&8@400&20@恢复精力的甘露.&103@0&104@0&105@&107@&108@0|1|3|铜钱|163.png|150
8|解毒丸|own|218.png|f_i_解毒丸&24@消耗品&25@99&20@解除中毒状态的药丸.&103@0&104@0&105@&107@&108@0|1|1|铜钱|163.png|10
`

const sourceWuliangMap2WeaponShopRows = `
0|尘沙剑|equip|55.png|captured|1|2|银元宝|39.png|10
1|伏魔棍|equip|56.png|captured|1|2|银元宝|39.png|10
2|破魄|equip|57.png|captured|1|2|银元宝|39.png|10
3|万相|equip|58.png|captured|1|2|银元宝|39.png|10
4|寒冰魄|equip|59.png|captured|1|2|银元宝|39.png|10
5|乾坤拳套|equip|60.png|captured|1|2|银元宝|39.png|10
6|流云法杖|equip|54.png|captured|1|2|银元宝|39.png|10
7|铁块|null|107.png|f_i_铁块^5BC46D&24@材料&25@99&20@用碎铁矿熔成的铁块&0;锻造用基本素材.&27@sitem_rock&103@0&104@0&105@&107@&108@69|1|1|铜钱|163.png|10|碎铁矿|105.png|10
8|铜块|null|108.png|f_i_铜块^5BC46D&24@材料&25@99&20@用铜钱熔成的铜块&0;锻造用中级素材.&27@sitem_rock&103@0&104@0&105@&107@&108@303|1|2|铜钱|163.png|10|铜钱|163.png|1000
9|银锭|null|109.png|f_i_银锭^5BC46D&24@材料&25@99&20@用银元宝熔成的银锭&0;锻造用中级素材.&27@sitem_rock&103@0&104@0&105@&107@&108@2000|1|2|铜钱|163.png|10|银元宝|39.png|10
`

const sourceWuliangMap2ArmorShopRows = `
0|龙颜钢盔|equip|378.png|captured|1|2|银元宝|39.png|2
1|龙颜重铠|equip|380.png|captured|1|2|银元宝|39.png|3
2|龙颜护腰|equip|383.png|captured|1|2|银元宝|39.png|1
3|龙颜护腿|equip|382.png|captured|1|2|银元宝|39.png|1
4|龙颜钢靴|equip|384.png|captured|1|2|银元宝|39.png|1
5|龙颜单肩|equip|379.png|captured|1|2|银元宝|39.png|1
6|龙颜护腕|equip|381.png|captured|1|2|银元宝|39.png|1
7|寒影面甲|equip|385.png|captured|1|2|银元宝|39.png|2
8|寒影锁甲|equip|387.png|captured|1|2|银元宝|39.png|3
9|寒影护腰|equip|390.png|captured|1|2|银元宝|39.png|1
10|寒影护腿|equip|389.png|captured|1|2|银元宝|39.png|1
11|寒影靴|equip|391.png|captured|1|2|银元宝|39.png|1
12|寒影护肩|equip|386.png|captured|1|2|银元宝|39.png|1
13|寒影护手|equip|388.png|captured|1|2|银元宝|39.png|1
14|流云之冠|equip|392.png|captured|1|2|银元宝|39.png|2
15|流云法衣|equip|394.png|captured|1|2|银元宝|39.png|3
16|流云护腰|equip|397.png|captured|1|2|银元宝|39.png|1
17|流云护腿|equip|396.png|captured|1|2|银元宝|39.png|1
18|流云之靴|equip|398.png|captured|1|2|银元宝|39.png|1
19|流云护肩|equip|393.png|captured|1|2|银元宝|39.png|1
20|流云护腕|equip|395.png|captured|1|2|银元宝|39.png|1
21|铁块|null|107.png|f_i_铁块^5BC46D&24@材料&25@99&20@用碎铁矿熔成的铁块&0;锻造用基本素材.&27@sitem_rock&103@0&104@0&105@&107@&108@69|1|1|铜钱|163.png|10|碎铁矿|105.png|10
22|铜块|null|108.png|f_i_铜块^5BC46D&24@材料&25@99&20@用铜钱熔成的铜块&0;锻造用中级素材.&27@sitem_rock&103@0&104@0&105@&107@&108@303|1|2|铜钱|163.png|10|铜钱|163.png|1000
23|银锭|null|109.png|f_i_银锭^5BC46D&24@材料&25@99&20@用银元宝熔成的银锭&0;锻造用中级素材.&27@sitem_rock&103@0&104@0&105@&107@&108@2000|1|2|铜钱|163.png|10|银元宝|39.png|10
`
