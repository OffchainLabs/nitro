;; Copyright 2022-2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
;; This file is auto-generated.

(module
    ;; symbols to re-export
    (import "user_host" "arbitrator_forward__read_args" (func $read_args (param i32)))
    (import "user_host" "arbitrator_forward__write_result" (func $write_result (param i32 i32)))
    (import "user_host" "arbitrator_forward__exit_early" (func $exit_early (param i32)))
    (import "user_host" "arbitrator_forward__storage_load_bytes32" (func $storage_load_bytes32 (param i32 i32)))
    (import "user_host" "arbitrator_forward__storage_cache_bytes32" (func $storage_cache_bytes32 (param i32 i32)))
    (import "user_host" "arbitrator_forward__storage_flush_cache" (func $storage_flush_cache (param i32)))
    (import "user_host" "arbitrator_forward__transient_load_bytes32" (func $transient_load_bytes32 (param i32 i32)))
    (import "user_host" "arbitrator_forward__transient_store_bytes32" (func $transient_store_bytes32 (param i32 i32)))
    (import "user_host" "arbitrator_forward__call_contract" (func $call_contract (param i32 i32 i32 i32 i64 i32) (result i32)))
    (import "user_host" "arbitrator_forward__delegate_call_contract" (func $delegate_call_contract (param i32 i32 i32 i64 i32) (result i32)))
    (import "user_host" "arbitrator_forward__static_call_contract" (func $static_call_contract (param i32 i32 i32 i64 i32) (result i32)))
    (import "user_host" "arbitrator_forward__create1" (func $create1 (param i32 i32 i32 i32 i32)))
    (import "user_host" "arbitrator_forward__create2" (func $create2 (param i32 i32 i32 i32 i32 i32)))
    (import "user_host" "arbitrator_forward__read_return_data" (func $read_return_data (param i32 i32 i32) (result i32)))
    (import "user_host" "arbitrator_forward__return_data_size" (func $return_data_size (result i32)))
    (import "user_host" "arbitrator_forward__emit_log" (func $emit_log (param i32 i32 i32)))
    (import "user_host" "arbitrator_forward__account_balance" (func $account_balance (param i32 i32)))
    (import "user_host" "arbitrator_forward__account_code" (func $account_code (param i32 i32 i32 i32) (result i32)))
    (import "user_host" "arbitrator_forward__account_code_size" (func $account_code_size (param i32) (result i32)))
    (import "user_host" "arbitrator_forward__account_codehash" (func $account_codehash (param i32 i32)))
    (import "user_host" "arbitrator_forward__evm_gas_left" (func $evm_gas_left (result i64)))
    (import "user_host" "arbitrator_forward__evm_ink_left" (func $evm_ink_left (result i64)))
    (import "user_host" "arbitrator_forward__block_basefee" (func $block_basefee (param i32)))
    (import "user_host" "arbitrator_forward__chainid" (func $chainid (result i64)))
    (import "user_host" "arbitrator_forward__block_coinbase" (func $block_coinbase (param i32)))
    (import "user_host" "arbitrator_forward__block_gas_limit" (func $block_gas_limit (result i64)))
    (import "user_host" "arbitrator_forward__block_number" (func $block_number (result i64)))
    (import "user_host" "arbitrator_forward__block_timestamp" (func $block_timestamp (result i64)))
    (import "user_host" "arbitrator_forward__contract_address" (func $contract_address (param i32)))
    (import "user_host" "arbitrator_forward__math_div" (func $math_div (param i32 i32)))
    (import "user_host" "arbitrator_forward__math_mod" (func $math_mod (param i32 i32)))
    (import "user_host" "arbitrator_forward__math_pow" (func $math_pow (param i32 i32)))
    (import "user_host" "arbitrator_forward__math_add_mod" (func $math_add_mod (param i32 i32 i32)))
    (import "user_host" "arbitrator_forward__math_mul_mod" (func $math_mul_mod (param i32 i32 i32)))
    (import "user_host" "arbitrator_forward__msg_reentrant" (func $msg_reentrant (result i32)))
    (import "user_host" "arbitrator_forward__msg_sender" (func $msg_sender (param i32)))
    (import "user_host" "arbitrator_forward__msg_value" (func $msg_value (param i32)))
    (import "user_host" "arbitrator_forward__native_keccak256" (func $native_keccak256 (param i32 i32 i32)))
    (import "user_host" "arbitrator_forward__tx_gas_price" (func $tx_gas_price (param i32)))
    (import "user_host" "arbitrator_forward__tx_ink_price" (func $tx_ink_price (result i32)))
    (import "user_host" "arbitrator_forward__tx_origin" (func $tx_origin (param i32)))
    (import "user_host" "arbitrator_forward__pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))

    ;; reserved offsets for future user_host imports
    (func $reserved_42 unreachable)
    (func $reserved_43 unreachable)
    (func $reserved_44 unreachable)
    (func $reserved_45 unreachable)
    (func $reserved_46 unreachable)
    (func $reserved_47 unreachable)
    (func $reserved_48 unreachable)
    (func $reserved_49 unreachable)
    (func $reserved_50 unreachable)
    (func $reserved_51 unreachable)
    (func $reserved_52 unreachable)
    (func $reserved_53 unreachable)
    (func $reserved_54 unreachable)
    (func $reserved_55 unreachable)
    (func $reserved_56 unreachable)
    (func $reserved_57 unreachable)
    (func $reserved_58 unreachable)
    (func $reserved_59 unreachable)
    (func $reserved_60 unreachable)
    (func $reserved_61 unreachable)
    (func $reserved_62 unreachable)
    (func $reserved_63 unreachable)
    (func $reserved_64 unreachable)
    (func $reserved_65 unreachable)
    (func $reserved_66 unreachable)
    (func $reserved_67 unreachable)
    (func $reserved_68 unreachable)
    (func $reserved_69 unreachable)
    (func $reserved_70 unreachable)
    (func $reserved_71 unreachable)
    (func $reserved_72 unreachable)
    (func $reserved_73 unreachable)
    (func $reserved_74 unreachable)
    (func $reserved_75 unreachable)
    (func $reserved_76 unreachable)
    (func $reserved_77 unreachable)
    (func $reserved_78 unreachable)
    (func $reserved_79 unreachable)
    (func $reserved_80 unreachable)
    (func $reserved_81 unreachable)
    (func $reserved_82 unreachable)
    (func $reserved_83 unreachable)
    (func $reserved_84 unreachable)
    (func $reserved_85 unreachable)
    (func $reserved_86 unreachable)
    (func $reserved_87 unreachable)
    (func $reserved_88 unreachable)
    (func $reserved_89 unreachable)
    (func $reserved_90 unreachable)
    (func $reserved_91 unreachable)
    (func $reserved_92 unreachable)
    (func $reserved_93 unreachable)
    (func $reserved_94 unreachable)
    (func $reserved_95 unreachable)
    (func $reserved_96 unreachable)
    (func $reserved_97 unreachable)
    (func $reserved_98 unreachable)
    (func $reserved_99 unreachable)
    (func $reserved_100 unreachable)
    (func $reserved_101 unreachable)
    (func $reserved_102 unreachable)
    (func $reserved_103 unreachable)
    (func $reserved_104 unreachable)
    (func $reserved_105 unreachable)
    (func $reserved_106 unreachable)
    (func $reserved_107 unreachable)
    (func $reserved_108 unreachable)
    (func $reserved_109 unreachable)
    (func $reserved_110 unreachable)
    (func $reserved_111 unreachable)
    (func $reserved_112 unreachable)
    (func $reserved_113 unreachable)
    (func $reserved_114 unreachable)
    (func $reserved_115 unreachable)
    (func $reserved_116 unreachable)
    (func $reserved_117 unreachable)
    (func $reserved_118 unreachable)
    (func $reserved_119 unreachable)
    (func $reserved_120 unreachable)
    (func $reserved_121 unreachable)
    (func $reserved_122 unreachable)
    (func $reserved_123 unreachable)
    (func $reserved_124 unreachable)
    (func $reserved_125 unreachable)
    (func $reserved_126 unreachable)
    (func $reserved_127 unreachable)
    (func $reserved_128 unreachable)
    (func $reserved_129 unreachable)
    (func $reserved_130 unreachable)
    (func $reserved_131 unreachable)
    (func $reserved_132 unreachable)
    (func $reserved_133 unreachable)
    (func $reserved_134 unreachable)
    (func $reserved_135 unreachable)
    (func $reserved_136 unreachable)
    (func $reserved_137 unreachable)
    (func $reserved_138 unreachable)
    (func $reserved_139 unreachable)
    (func $reserved_140 unreachable)
    (func $reserved_141 unreachable)
    (func $reserved_142 unreachable)
    (func $reserved_143 unreachable)
    (func $reserved_144 unreachable)
    (func $reserved_145 unreachable)
    (func $reserved_146 unreachable)
    (func $reserved_147 unreachable)
    (func $reserved_148 unreachable)
    (func $reserved_149 unreachable)
    (func $reserved_150 unreachable)
    (func $reserved_151 unreachable)
    (func $reserved_152 unreachable)
    (func $reserved_153 unreachable)
    (func $reserved_154 unreachable)
    (func $reserved_155 unreachable)
    (func $reserved_156 unreachable)
    (func $reserved_157 unreachable)
    (func $reserved_158 unreachable)
    (func $reserved_159 unreachable)
    (func $reserved_160 unreachable)
    (func $reserved_161 unreachable)
    (func $reserved_162 unreachable)
    (func $reserved_163 unreachable)
    (func $reserved_164 unreachable)
    (func $reserved_165 unreachable)
    (func $reserved_166 unreachable)
    (func $reserved_167 unreachable)
    (func $reserved_168 unreachable)
    (func $reserved_169 unreachable)
    (func $reserved_170 unreachable)
    (func $reserved_171 unreachable)
    (func $reserved_172 unreachable)
    (func $reserved_173 unreachable)
    (func $reserved_174 unreachable)
    (func $reserved_175 unreachable)
    (func $reserved_176 unreachable)
    (func $reserved_177 unreachable)
    (func $reserved_178 unreachable)
    (func $reserved_179 unreachable)
    (func $reserved_180 unreachable)
    (func $reserved_181 unreachable)
    (func $reserved_182 unreachable)
    (func $reserved_183 unreachable)
    (func $reserved_184 unreachable)
    (func $reserved_185 unreachable)
    (func $reserved_186 unreachable)
    (func $reserved_187 unreachable)
    (func $reserved_188 unreachable)
    (func $reserved_189 unreachable)
    (func $reserved_190 unreachable)
    (func $reserved_191 unreachable)
    (func $reserved_192 unreachable)
    (func $reserved_193 unreachable)
    (func $reserved_194 unreachable)
    (func $reserved_195 unreachable)
    (func $reserved_196 unreachable)
    (func $reserved_197 unreachable)
    (func $reserved_198 unreachable)
    (func $reserved_199 unreachable)
    (func $reserved_200 unreachable)
    (func $reserved_201 unreachable)
    (func $reserved_202 unreachable)
    (func $reserved_203 unreachable)
    (func $reserved_204 unreachable)
    (func $reserved_205 unreachable)
    (func $reserved_206 unreachable)
    (func $reserved_207 unreachable)
    (func $reserved_208 unreachable)
    (func $reserved_209 unreachable)
    (func $reserved_210 unreachable)
    (func $reserved_211 unreachable)
    (func $reserved_212 unreachable)
    (func $reserved_213 unreachable)
    (func $reserved_214 unreachable)
    (func $reserved_215 unreachable)
    (func $reserved_216 unreachable)
    (func $reserved_217 unreachable)
    (func $reserved_218 unreachable)
    (func $reserved_219 unreachable)
    (func $reserved_220 unreachable)
    (func $reserved_221 unreachable)
    (func $reserved_222 unreachable)
    (func $reserved_223 unreachable)
    (func $reserved_224 unreachable)
    (func $reserved_225 unreachable)
    (func $reserved_226 unreachable)
    (func $reserved_227 unreachable)
    (func $reserved_228 unreachable)
    (func $reserved_229 unreachable)
    (func $reserved_230 unreachable)
    (func $reserved_231 unreachable)
    (func $reserved_232 unreachable)
    (func $reserved_233 unreachable)
    (func $reserved_234 unreachable)
    (func $reserved_235 unreachable)
    (func $reserved_236 unreachable)
    (func $reserved_237 unreachable)
    (func $reserved_238 unreachable)
    (func $reserved_239 unreachable)
    (func $reserved_240 unreachable)
    (func $reserved_241 unreachable)
    (func $reserved_242 unreachable)
    (func $reserved_243 unreachable)
    (func $reserved_244 unreachable)
    (func $reserved_245 unreachable)
    (func $reserved_246 unreachable)
    (func $reserved_247 unreachable)
    (func $reserved_248 unreachable)
    (func $reserved_249 unreachable)
    (func $reserved_250 unreachable)
    (func $reserved_251 unreachable)
    (func $reserved_252 unreachable)
    (func $reserved_253 unreachable)
    (func $reserved_254 unreachable)
    (func $reserved_255 unreachable)
    (func $reserved_256 unreachable)
    (func $reserved_257 unreachable)
    (func $reserved_258 unreachable)
    (func $reserved_259 unreachable)
    (func $reserved_260 unreachable)
    (func $reserved_261 unreachable)
    (func $reserved_262 unreachable)
    (func $reserved_263 unreachable)
    (func $reserved_264 unreachable)
    (func $reserved_265 unreachable)
    (func $reserved_266 unreachable)
    (func $reserved_267 unreachable)
    (func $reserved_268 unreachable)
    (func $reserved_269 unreachable)
    (func $reserved_270 unreachable)
    (func $reserved_271 unreachable)
    (func $reserved_272 unreachable)
    (func $reserved_273 unreachable)
    (func $reserved_274 unreachable)
    (func $reserved_275 unreachable)
    (func $reserved_276 unreachable)
    (func $reserved_277 unreachable)
    (func $reserved_278 unreachable)
    (func $reserved_279 unreachable)
    (func $reserved_280 unreachable)
    (func $reserved_281 unreachable)
    (func $reserved_282 unreachable)
    (func $reserved_283 unreachable)
    (func $reserved_284 unreachable)
    (func $reserved_285 unreachable)
    (func $reserved_286 unreachable)
    (func $reserved_287 unreachable)
    (func $reserved_288 unreachable)
    (func $reserved_289 unreachable)
    (func $reserved_290 unreachable)
    (func $reserved_291 unreachable)
    (func $reserved_292 unreachable)
    (func $reserved_293 unreachable)
    (func $reserved_294 unreachable)
    (func $reserved_295 unreachable)
    (func $reserved_296 unreachable)
    (func $reserved_297 unreachable)
    (func $reserved_298 unreachable)
    (func $reserved_299 unreachable)
    (func $reserved_300 unreachable)
    (func $reserved_301 unreachable)
    (func $reserved_302 unreachable)
    (func $reserved_303 unreachable)
    (func $reserved_304 unreachable)
    (func $reserved_305 unreachable)
    (func $reserved_306 unreachable)
    (func $reserved_307 unreachable)
    (func $reserved_308 unreachable)
    (func $reserved_309 unreachable)
    (func $reserved_310 unreachable)
    (func $reserved_311 unreachable)
    (func $reserved_312 unreachable)
    (func $reserved_313 unreachable)
    (func $reserved_314 unreachable)
    (func $reserved_315 unreachable)
    (func $reserved_316 unreachable)
    (func $reserved_317 unreachable)
    (func $reserved_318 unreachable)
    (func $reserved_319 unreachable)
    (func $reserved_320 unreachable)
    (func $reserved_321 unreachable)
    (func $reserved_322 unreachable)
    (func $reserved_323 unreachable)
    (func $reserved_324 unreachable)
    (func $reserved_325 unreachable)
    (func $reserved_326 unreachable)
    (func $reserved_327 unreachable)
    (func $reserved_328 unreachable)
    (func $reserved_329 unreachable)
    (func $reserved_330 unreachable)
    (func $reserved_331 unreachable)
    (func $reserved_332 unreachable)
    (func $reserved_333 unreachable)
    (func $reserved_334 unreachable)
    (func $reserved_335 unreachable)
    (func $reserved_336 unreachable)
    (func $reserved_337 unreachable)
    (func $reserved_338 unreachable)
    (func $reserved_339 unreachable)
    (func $reserved_340 unreachable)
    (func $reserved_341 unreachable)
    (func $reserved_342 unreachable)
    (func $reserved_343 unreachable)
    (func $reserved_344 unreachable)
    (func $reserved_345 unreachable)
    (func $reserved_346 unreachable)
    (func $reserved_347 unreachable)
    (func $reserved_348 unreachable)
    (func $reserved_349 unreachable)
    (func $reserved_350 unreachable)
    (func $reserved_351 unreachable)
    (func $reserved_352 unreachable)
    (func $reserved_353 unreachable)
    (func $reserved_354 unreachable)
    (func $reserved_355 unreachable)
    (func $reserved_356 unreachable)
    (func $reserved_357 unreachable)
    (func $reserved_358 unreachable)
    (func $reserved_359 unreachable)
    (func $reserved_360 unreachable)
    (func $reserved_361 unreachable)
    (func $reserved_362 unreachable)
    (func $reserved_363 unreachable)
    (func $reserved_364 unreachable)
    (func $reserved_365 unreachable)
    (func $reserved_366 unreachable)
    (func $reserved_367 unreachable)
    (func $reserved_368 unreachable)
    (func $reserved_369 unreachable)
    (func $reserved_370 unreachable)
    (func $reserved_371 unreachable)
    (func $reserved_372 unreachable)
    (func $reserved_373 unreachable)
    (func $reserved_374 unreachable)
    (func $reserved_375 unreachable)
    (func $reserved_376 unreachable)
    (func $reserved_377 unreachable)
    (func $reserved_378 unreachable)
    (func $reserved_379 unreachable)
    (func $reserved_380 unreachable)
    (func $reserved_381 unreachable)
    (func $reserved_382 unreachable)
    (func $reserved_383 unreachable)
    (func $reserved_384 unreachable)
    (func $reserved_385 unreachable)
    (func $reserved_386 unreachable)
    (func $reserved_387 unreachable)
    (func $reserved_388 unreachable)
    (func $reserved_389 unreachable)
    (func $reserved_390 unreachable)
    (func $reserved_391 unreachable)
    (func $reserved_392 unreachable)
    (func $reserved_393 unreachable)
    (func $reserved_394 unreachable)
    (func $reserved_395 unreachable)
    (func $reserved_396 unreachable)
    (func $reserved_397 unreachable)
    (func $reserved_398 unreachable)
    (func $reserved_399 unreachable)
    (func $reserved_400 unreachable)
    (func $reserved_401 unreachable)
    (func $reserved_402 unreachable)
    (func $reserved_403 unreachable)
    (func $reserved_404 unreachable)
    (func $reserved_405 unreachable)
    (func $reserved_406 unreachable)
    (func $reserved_407 unreachable)
    (func $reserved_408 unreachable)
    (func $reserved_409 unreachable)
    (func $reserved_410 unreachable)
    (func $reserved_411 unreachable)
    (func $reserved_412 unreachable)
    (func $reserved_413 unreachable)
    (func $reserved_414 unreachable)
    (func $reserved_415 unreachable)
    (func $reserved_416 unreachable)
    (func $reserved_417 unreachable)
    (func $reserved_418 unreachable)
    (func $reserved_419 unreachable)
    (func $reserved_420 unreachable)
    (func $reserved_421 unreachable)
    (func $reserved_422 unreachable)
    (func $reserved_423 unreachable)
    (func $reserved_424 unreachable)
    (func $reserved_425 unreachable)
    (func $reserved_426 unreachable)
    (func $reserved_427 unreachable)
    (func $reserved_428 unreachable)
    (func $reserved_429 unreachable)
    (func $reserved_430 unreachable)
    (func $reserved_431 unreachable)
    (func $reserved_432 unreachable)
    (func $reserved_433 unreachable)
    (func $reserved_434 unreachable)
    (func $reserved_435 unreachable)
    (func $reserved_436 unreachable)
    (func $reserved_437 unreachable)
    (func $reserved_438 unreachable)
    (func $reserved_439 unreachable)
    (func $reserved_440 unreachable)
    (func $reserved_441 unreachable)
    (func $reserved_442 unreachable)
    (func $reserved_443 unreachable)
    (func $reserved_444 unreachable)
    (func $reserved_445 unreachable)
    (func $reserved_446 unreachable)
    (func $reserved_447 unreachable)
    (func $reserved_448 unreachable)
    (func $reserved_449 unreachable)
    (func $reserved_450 unreachable)
    (func $reserved_451 unreachable)
    (func $reserved_452 unreachable)
    (func $reserved_453 unreachable)
    (func $reserved_454 unreachable)
    (func $reserved_455 unreachable)
    (func $reserved_456 unreachable)
    (func $reserved_457 unreachable)
    (func $reserved_458 unreachable)
    (func $reserved_459 unreachable)
    (func $reserved_460 unreachable)
    (func $reserved_461 unreachable)
    (func $reserved_462 unreachable)
    (func $reserved_463 unreachable)
    (func $reserved_464 unreachable)
    (func $reserved_465 unreachable)
    (func $reserved_466 unreachable)
    (func $reserved_467 unreachable)
    (func $reserved_468 unreachable)
    (func $reserved_469 unreachable)
    (func $reserved_470 unreachable)
    (func $reserved_471 unreachable)
    (func $reserved_472 unreachable)
    (func $reserved_473 unreachable)
    (func $reserved_474 unreachable)
    (func $reserved_475 unreachable)
    (func $reserved_476 unreachable)
    (func $reserved_477 unreachable)
    (func $reserved_478 unreachable)
    (func $reserved_479 unreachable)
    (func $reserved_480 unreachable)
    (func $reserved_481 unreachable)
    (func $reserved_482 unreachable)
    (func $reserved_483 unreachable)
    (func $reserved_484 unreachable)
    (func $reserved_485 unreachable)
    (func $reserved_486 unreachable)
    (func $reserved_487 unreachable)
    (func $reserved_488 unreachable)
    (func $reserved_489 unreachable)
    (func $reserved_490 unreachable)
    (func $reserved_491 unreachable)
    (func $reserved_492 unreachable)
    (func $reserved_493 unreachable)
    (func $reserved_494 unreachable)
    (func $reserved_495 unreachable)
    (func $reserved_496 unreachable)
    (func $reserved_497 unreachable)
    (func $reserved_498 unreachable)
    (func $reserved_499 unreachable)
    (func $reserved_500 unreachable)
    (func $reserved_501 unreachable)
    (func $reserved_502 unreachable)
    (func $reserved_503 unreachable)
    (func $reserved_504 unreachable)
    (func $reserved_505 unreachable)
    (func $reserved_506 unreachable)
    (func $reserved_507 unreachable)
    (func $reserved_508 unreachable)
    (func $reserved_509 unreachable)
    (func $reserved_510 unreachable)
    (func $reserved_511 unreachable)

    ;; allows user_host to request a trap
    (global $trap (mut i32) (i32.const 0))
    (func $check
        global.get $trap                    ;; see if set
        (global.set $trap (i32.const 0))    ;; reset the flag
        (if (then (unreachable)))
    )
    (func (export "forward__set_trap")
        (global.set $trap (i32.const 1))
    )

    ;; user linkage
    (func (export "vm_hooks__read_args") (param i32)
        local.get 0
        call $read_args
        call $check
    )
    (func (export "vm_hooks__write_result") (param i32 i32)
        local.get 0
        local.get 1
        call $write_result
        call $check
    )
    (func (export "vm_hooks__exit_early") (param i32)
        local.get 0
        call $exit_early
        call $check
    )
    (func (export "vm_hooks__storage_load_bytes32") (param i32 i32)
        local.get 0
        local.get 1
        call $storage_load_bytes32
        call $check
    )
    (func (export "vm_hooks__storage_cache_bytes32") (param i32 i32)
        local.get 0
        local.get 1
        call $storage_cache_bytes32
        call $check
    )
    (func (export "vm_hooks__storage_flush_cache") (param i32)
        local.get 0
        call $storage_flush_cache
        call $check
    )
    (func (export "vm_hooks__transient_load_bytes32") (param i32 i32)
        local.get 0
        local.get 1
        call $transient_load_bytes32
        call $check
    )
    (func (export "vm_hooks__transient_store_bytes32") (param i32 i32)
        local.get 0
        local.get 1
        call $transient_store_bytes32
        call $check
    )
    (func (export "vm_hooks__call_contract") (param i32 i32 i32 i32 i64 i32) (result i32)
        local.get 0
        local.get 1
        local.get 2
        local.get 3
        local.get 4
        local.get 5
        call $call_contract
        call $check
    )
    (func (export "vm_hooks__delegate_call_contract") (param i32 i32 i32 i64 i32) (result i32)
        local.get 0
        local.get 1
        local.get 2
        local.get 3
        local.get 4
        call $delegate_call_contract
        call $check
    )
    (func (export "vm_hooks__static_call_contract") (param i32 i32 i32 i64 i32) (result i32)
        local.get 0
        local.get 1
        local.get 2
        local.get 3
        local.get 4
        call $static_call_contract
        call $check
    )
    (func (export "vm_hooks__create1") (param i32 i32 i32 i32 i32)
        local.get 0
        local.get 1
        local.get 2
        local.get 3
        local.get 4
        call $create1
        call $check
    )
    (func (export "vm_hooks__create2") (param i32 i32 i32 i32 i32 i32)
        local.get 0
        local.get 1
        local.get 2
        local.get 3
        local.get 4
        local.get 5
        call $create2
        call $check
    )
    (func (export "vm_hooks__read_return_data") (param i32 i32 i32) (result i32)
        local.get 0
        local.get 1
        local.get 2
        call $read_return_data
        call $check
    )
    (func (export "vm_hooks__return_data_size") (result i32)
        call $return_data_size
        call $check
    )
    (func (export "vm_hooks__emit_log") (param i32 i32 i32)
        local.get 0
        local.get 1
        local.get 2
        call $emit_log
        call $check
    )
    (func (export "vm_hooks__account_balance") (param i32 i32)
        local.get 0
        local.get 1
        call $account_balance
        call $check
    )
    (func (export "vm_hooks__account_code") (param i32 i32 i32 i32) (result i32)
        local.get 0
        local.get 1
        local.get 2
        local.get 3
        call $account_code
        call $check
    )
    (func (export "vm_hooks__account_code_size") (param i32) (result i32)
        local.get 0
        call $account_code_size
        call $check
    )
    (func (export "vm_hooks__account_codehash") (param i32 i32)
        local.get 0
        local.get 1
        call $account_codehash
        call $check
    )
    (func (export "vm_hooks__evm_gas_left") (result i64)
        call $evm_gas_left
        call $check
    )
    (func (export "vm_hooks__evm_ink_left") (result i64)
        call $evm_ink_left
        call $check
    )
    (func (export "vm_hooks__block_basefee") (param i32)
        local.get 0
        call $block_basefee
        call $check
    )
    (func (export "vm_hooks__chainid") (result i64)
        call $chainid
        call $check
    )
    (func (export "vm_hooks__block_coinbase") (param i32)
        local.get 0
        call $block_coinbase
        call $check
    )
    (func (export "vm_hooks__block_gas_limit") (result i64)
        call $block_gas_limit
        call $check
    )
    (func (export "vm_hooks__block_number") (result i64)
        call $block_number
        call $check
    )
    (func (export "vm_hooks__block_timestamp") (result i64)
        call $block_timestamp
        call $check
    )
    (func (export "vm_hooks__contract_address") (param i32)
        local.get 0
        call $contract_address
        call $check
    )
    (func (export "vm_hooks__math_div") (param i32 i32)
        local.get 0
        local.get 1
        call $math_div
        call $check
    )
    (func (export "vm_hooks__math_mod") (param i32 i32)
        local.get 0
        local.get 1
        call $math_mod
        call $check
    )
    (func (export "vm_hooks__math_pow") (param i32 i32)
        local.get 0
        local.get 1
        call $math_pow
        call $check
    )
    (func (export "vm_hooks__math_add_mod") (param i32 i32 i32)
        local.get 0
        local.get 1
        local.get 2
        call $math_add_mod
        call $check
    )
    (func (export "vm_hooks__math_mul_mod") (param i32 i32 i32)
        local.get 0
        local.get 1
        local.get 2
        call $math_mul_mod
        call $check
    )
    (func (export "vm_hooks__msg_reentrant") (result i32)
        call $msg_reentrant
        call $check
    )
    (func (export "vm_hooks__msg_sender") (param i32)
        local.get 0
        call $msg_sender
        call $check
    )
    (func (export "vm_hooks__msg_value") (param i32)
        local.get 0
        call $msg_value
        call $check
    )
    (func (export "vm_hooks__native_keccak256") (param i32 i32 i32)
        local.get 0
        local.get 1
        local.get 2
        call $native_keccak256
        call $check
    )
    (func (export "vm_hooks__tx_gas_price") (param i32)
        local.get 0
        call $tx_gas_price
        call $check
    )
    (func (export "vm_hooks__tx_ink_price") (result i32)
        call $tx_ink_price
        call $check
    )
    (func (export "vm_hooks__tx_origin") (param i32)
        local.get 0
        call $tx_origin
        call $check
    )
    (func (export "vm_hooks__pay_for_memory_grow") (param i32)
        local.get 0
        call $pay_for_memory_grow
        call $check
    )
)
