# Multi-Dimensional Gas (MultiGas) Refunds

## Context and Problem Statement

As part of the multi-dimensional constraint-based pricing, Arbitrum can price different resources according to their network usage. This allows charging more for transactions that use constrained resources and less for transactions that use available resources. However, Arbitrum still uses the L1-Ethereum transaction types, which don't provide a way for users to specify multi-dimensional gas and base-fees. Instead, the existing transaction type only has a single-dimensional gas amount (GasLimit) and a single-dimensional base-fee (MaxFeePerGas).

## Decision Outcome

When multi-gas constraints are enabled, the Arbitrum chain keeps a base-fee for each resource dimension. If the L2-gas model needs to constrain a specific resource dimension, it raises the base fee of this resource without increasing the other ones. Additionally, to be compatible with Ethereum, Arbitrum sets the single-dimensional base fee as the maximum base fee among all resources.

Even with multi-gas, Arbitrum charges gas in the beginning of a transaction by multiplying the provided single-dimensional gas limit by the single-dimensional base fee. It does that to preserve the existing Ethereum transaction API. In the beginning of the transaction, Arbitrum doesn't know how much of each resource will be used because this is not available in the existing transaction API.

After executing the transaction, Arbitrum knows how much of each resource was used and the base-fees for each resource. So, Arbitrum computes the discounted price of a transaction by multiplying each used gas resource by its corresponding base fee. If none of the resources are being constrained, the discounted price equals the initial price. However, when a resource is being constrained more than the other ones, the discounted price will be less than the initial price. At this point, Arbitrum computes the difference between the discounted price and the initial price, and gives a refund to the user.

The SpecialFee resource kind is a particular case for refunds. Some places of Arbitrum charge an L2-only fee as gas. For instance, Arbitrum charges gas to pay for the L1 cost of posting a transaction, and it charges gas for paying the execution of a future transaction (called retryable). Arbitrum charges these fees as gas instead of one-off transfers to preserve the existing Ethereum transaction API. Those fees don't correspond to any machine-specific resource, so we created a special dimension for them.

When computing the amount of SpecialFee gas that should be charged, Arbitrum divides the fee amount by the single-gas base fee, resulting in a gas amount. The base fee used in this case is the maximum base fee among all resources. So, the SpecialFee base fee must always be equal to the maximum base fee. Otherwise, there would be a risk of giving an unintended refund when charging a special fee. We must not give multi-gas refunds to the SpecialFee resource dimension.
