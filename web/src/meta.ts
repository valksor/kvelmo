/**
 * Single source of truth for the tool name and derived values.
 * Change `name` here to rebrand the entire web frontend.
 */

export const name = 'kvelmo'

/** Returns a Zustand store name: kvelmo-{suffix} */
export const storeName = (suffix: string) => `${name}-${suffix}`

/** Returns an environment variable name: KVELMO_{SUFFIX} */
export const envVar = (suffix: string) => `${name.toUpperCase()}_${suffix}`

/** Returns a quoted CLI command reference: 'kvelmo {sub}' */
export const cliCmd = (sub: string) => `'${name} ${sub}'`
