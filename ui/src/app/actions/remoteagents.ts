'use server'
import { RemoteAgent, BaseResponse } from "@/types";
import { fetchApi, createErrorResponse } from "./utils";
import { revalidatePath } from "next/cache";

export async function getRemoteAgents(): Promise<BaseResponse<RemoteAgent[]>> {
  try {
    const response = await fetchApi<BaseResponse<RemoteAgent[]>>(`/remoteagents`);
    if (!response) {
      throw new Error("Failed to get RemoteAgents");
    }
    return {
      message: "RemoteAgents fetched successfully",
      data: response.data,
    };
  } catch (error) {
    return createErrorResponse<RemoteAgent[]>(error, "Error getting RemoteAgents");
  }
}

export async function createRemoteAgent(remoteAgent: RemoteAgent): Promise<BaseResponse<RemoteAgent>> {
  try {
    const response = await fetchApi<BaseResponse<RemoteAgent>>("/remoteagents", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(remoteAgent),
    });
    revalidatePath("/agents");
    return response;
  } catch (error) {
    return createErrorResponse<RemoteAgent>(error, "Error creating RemoteAgent");
  }
}

export async function deleteRemoteAgent(namespace: string, name: string): Promise<BaseResponse<void>> {
  try {
    await fetchApi<BaseResponse<void>>(`/remoteagents/${namespace}/${name}`, {
      method: "DELETE",
    });
    revalidatePath("/agents");
    return { message: "RemoteAgent deleted successfully" };
  } catch (error) {
    return createErrorResponse<void>(error, "Error deleting RemoteAgent");
  }
}
