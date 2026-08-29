package frontendgen

import "strings"

// vbenComposableCode 生成 api/composables/*.ts（对应 TS 版 composable-template.ts）
// 直接使用 #/api/client 的 apiClient 单例调用 Service Client。
func vbenComposableCode(service *ParsedService, serviceName string) string {
	crudPaths := GetCrudPaths(service)
	hasList := crudPaths.List != nil
	hasGet := crudPaths.Get != nil
	hasCreate := crudPaths.Create != nil
	hasUpdate := crudPaths.Update != nil
	hasDelete := crudPaths.Delete != nil

	modelPascal := toPascalCase(service.ModelName)
	prefix := service.TypePrefix
	client := "apiClient." + service.ClientGetterName

	var typeImports []string
	if hasDelete {
		typeImports = append(typeImports, prefix+"_Delete"+modelPascal+"Request")
	}
	if hasGet {
		typeImports = append(typeImports, prefix+"_Get"+modelPascal+"Request")
	}
	if hasList {
		typeImports = append(typeImports, prefix+"_List"+modelPascal+"Response")
	}
	if hasGet || hasList {
		typeImports = append(typeImports, prefix+"_"+modelPascal)
	}

	var sb strings.Builder
	sb.WriteString("import type {\n")
	for _, imp := range typeImports {
		sb.WriteString("  " + imp + ",\n")
	}
	sb.WriteString("} from '#/api/generated/" + serviceName + "/service/v1';\n\n")

	sb.WriteString(`import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

`)

	sb.WriteString("import { apiClient } from '#/api/client';\n")
	sb.WriteString("import { queryClient } from '#/plugins/vue-query';\n")
	sb.WriteString("import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';\n\n")

	sb.WriteString("// ==============================\n")
	sb.WriteString("// " + service.ModelName + " 管理\n")
	sb.WriteString("// ==============================\n\n")

	if hasList {
		sb.WriteString(`export function useList` + modelPascal + `s(
  query: PaginationQuery,
  options?: UseQueryOptions<` + prefix + `_List` + modelPascal + `Response, Error>,
) {
  return useQuery({
    queryKey: ['list` + modelPascal + `s', query],
    queryFn: () => ` + client + `.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchList` + modelPascal + `s(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['list` + modelPascal + `s', params],
    queryFn: () => ` + client + `.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

`)
	}

	if hasGet {
		sb.WriteString(`export function useGet` + modelPascal + `(
  req: ` + prefix + `_Get` + modelPascal + `Request,
  options?: UseQueryOptions<` + prefix + `_` + modelPascal + `, Error>,
) {
  return useQuery({
    queryKey: ['get` + modelPascal + `', req],
    queryFn: () => ` + client + `.Get(req),
    ...options,
  });
}

`)
	}

	if hasCreate {
		sb.WriteString(`export function useCreate` + modelPascal + `(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      ` + client + `.Create({ data: { ...values } as ` + prefix + `_` + modelPascal + ` }),
    ...options,
  });
}

`)
	}

	if hasUpdate {
		sb.WriteString(`export function useUpdate` + modelPascal + `(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
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
		sb.WriteString(`export function useDelete` + modelPascal + `(
  options?: UseMutationOptions<
    object,
    Error,
    ` + prefix + `_Delete` + modelPascal + `Request
  >,
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
