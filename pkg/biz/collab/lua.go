package collab

// luaAppend 原子执行：INCR 分配序号 + XADD 写入 Stream。
// KEYS[1] = seq_key
// KEYS[2] = stream_key
// ARGV[1] = max_len (Stream MAXLEN)
// ARGV[2] = payload (序列化后的事件 JSON)
//
// 返回分配的 seq。
const luaAppend = `
local seq_key = KEYS[1]
local stream_key = KEYS[2]
local max_len = tonumber(ARGV[1])
local payload = ARGV[2]

local seq = redis.call('INCR', seq_key)
redis.call('XADD', stream_key, 'MAXLEN', '~', max_len, '*', 'seq', tostring(seq), 'data', payload)
return seq
`
