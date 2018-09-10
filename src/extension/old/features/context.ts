import { Context } from '../../../environment/context/context'
import { MessageConnection } from '../../../jsonrpc2/connection'
import { ContextUpdateNotification, ContextUpdateParams } from '../../../protocol/context'
import { ExtensionContext } from '../api'

/**
 * Creates the Sourcegraph extension API's {@link SourcegraphExtensionAPI#context} value.
 *
 * @param rawConnection The connection to the Sourcegraph API client.
 * @return The {@link SourcegraphExtensionAPI#context} value.
 */
export function createExtContext(rawConnection: MessageConnection): ExtensionContext {
    return {
        updateContext: (updates: Context): void => {
            rawConnection.sendNotification(ContextUpdateNotification.type, { updates } as ContextUpdateParams)
        },
    }
}
