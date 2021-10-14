# Burstable

## 实现

- Quota：基线可用量
- Period：使用计量周期
- Burst：周期内允许的最大突发值
- Burst Credit：累计的可用突发值
- Max：周期内允许的最大的最终值

公式：
- 下一轮可用 = Quota + min(Burst, Burst Credit)
- Burst Credit = Quota - 本轮可用 + 上一轮 Burst Credit