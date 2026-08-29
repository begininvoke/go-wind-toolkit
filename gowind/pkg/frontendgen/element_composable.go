package frontendgen

import "strings"

// elementComposableCode 生成 api/composables/*.ts（对应 TS 版 composable-template.ts）
// 直接使用 @/api/client 的 apiClient 单例调用 Service Client。
func elementComposableCode(service *ParsedService, serviceName string) string {
	crudPaths := GetCrudPaths(service)
	hasList := crudPaths.List != nil
	hasGet := crudPaths.Get != nil
	hasCreate := crudPaths.Create != nil
	hasUpdate := crudPaths.Update != nil
	hasDelete := crudPaths.Delete != nil

	modelPascal := toPascalCase(service.ModelName)
	prefix := service.TypePrefix
	client := "apiClient." + service.ClientGetterName

	var sb strings.Builder

	sb.WriteString(`import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
`)

	var typeImports []string
	if hasList {
		typeImports = append(typeImports, prefix+"_List"+modelPascal+"Response")
	}
	if hasGet {
		typeImports = append(typeImports, prefix+"_"+modelPascal)
	}
	if hasDelete {
		typeImports = append(typeImports, prefix+"_Delete"+modelPascal+"Request")
	}
	if len(typeImports) > 0 {
		sb.WriteString("import type {\n")
		for _, t := range typeImports {
			sb.WriteString("  " + t + ",\n")
		}
		sb.WriteString("} from \"@/api/generated/" + serviceName + "/service/v1\";\n")
	}

	sb.WriteString(`import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";

`)

	if hasList {
		sb.WriteString(`export function useList` + modelPascal + `s(
  query: PaginationQuery,
  options?: UseQueryOptions<` + prefix + `_List` + modelPascal + `Response, Error>
) {
  return useQuery({
    queryKey: ["list` + modelPascal + `s", query],
    queryFn: () => ` + client + `.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchList` + modelPascal + `s(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["list` + modelPascal + `s", params],
    queryFn: () => ` + client + `.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}
`)
	}

	if hasGet {
		sb.WriteString(`
export function useGet` + modelPascal + `(
  req: ` + prefix + `_Get` + modelPascal + `Request,
  options?: UseQueryOptions<` + prefix + `_` + modelPascal + `, Error>
) {
  return useQuery({
    queryKey: ["get` + modelPascal + `", req],
    queryFn: () => ` + client + `.Get(req),
    ...options,
  });
}
`)
	}

	if hasCreate {
		sb.WriteString(`
export function useCreate` + modelPascal + `(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) =>
      ` + client + `.Create({ data: { ...values } as ` + prefix + `_` + modelPascal + ` }),
    ...options,
  });
}
`)
	}

	if hasUpdate {
		sb.WriteString(`
export function useUpdate` + modelPascal + `(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      ` + client + `.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}
`)
	}

	if hasDelete {
		sb.WriteString(`
export function useDelete` + modelPascal + `(
  options?: UseMutationOptions<{}, Error, ` + prefix + `_Delete` + modelPascal + `Request>
) {
  return useMutation({
    mutationFn: (req) => ` + client + `.Delete(req),
    ...options,
  });
}
`)
	}

	return sb.String()
}
