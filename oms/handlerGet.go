// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"net/http"

	"github.com/openmpp/go/ompp/db"
	"github.com/openmpp/go/ompp/omppLog"
)

// modelListHandler return list of models as model_dic rows and model directory:
// GET /api/model-list
func modelListHandler(w http.ResponseWriter, r *http.Request) {

	// get list of model basics: name, digest and directory
	mbs := theCatalog.allModels()

	type modelListItem struct {
		Model db.ModelDicRow // model_dic db row
		Dir   string         // model directory, relative to model root and slashed: dir/sub
		Extra string         // if not empty then model extra content
	}
	ml := make([]modelListItem, 0, len(mbs))

	// by model digest get model_dic row
	for _, b := range mbs {
		if m, ok := theCatalog.ModelDicByDigest(b.digest); ok {
			ml = append(ml,
				modelListItem{
					Model: m,
					Dir:   b.relDir,
					Extra: b.extra,
				})
		}
	}

	// write json response
	jsonResponse(w, r, ml)
}

// modelTextListHandler return list models as model_dic row, model_dic_txt row and model directory:
// GET /api/model-list/text
// GET /api/model-list/text/lang/:lang
// If optional lang specified then result in that language else in browser language or model default.
func modelTextListHandler(w http.ResponseWriter, r *http.Request) {

	rqLangTags := getRequestLang(r, "lang") // get optional language argument and languages accepted by browser

	// get list of model basics: name, digest and directory
	mbs := theCatalog.allModels()

	type modelTxtListItem struct {
		ModelDicDescrNote        // model_dic db row and model_dic_txt row
		Dir               string // model directory, relative to model root and slashed: dir/sub
		Extra             string // if not empty then model extra content
	}
	mtl := make([]modelTxtListItem, 0, len(mbs))

	// by model digest get model_dic row and model_dic_txt row in UI language
	for _, b := range mbs {
		if mt, ok := theCatalog.ModelTextByDigest(b.digest, rqLangTags); ok {
			mtl = append(mtl,
				modelTxtListItem{
					ModelDicDescrNote: *mt,
					Dir:               b.relDir,
					Extra:             b.extra,
				})
		}
	}

	// write json response
	jsonResponse(w, r, mtl)
}

// modelMetaHandler return language-indepedent model metadata:
// GET /api/model/:model
// If multiple models with same name exist only one is returned.
func modelMetaHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")

	m, _ := theCatalog.ModelMetaByDigestOrName(dn)
	jsonResponse(w, r, m)
}

// modelTextHandler return language-specific model metadata:
// GET /api/model/:model/text
// GET /api/model/:model/text/lang/:lang
// Model digest-or-name must specified, if multiple models with same name exist only one is returned.
// If optional lang specified then result in that language else in browser language or model default.
func modelTextHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	rqLangTags := getRequestLang(r, "lang") // get optional language argument and languages accepted by browser

	mt, _ := theCatalog.ModelMetaTextByDigestOrName(dn, rqLangTags)
	jsonResponse(w, r, mt)
}

// modelAllTextHandler return language-specific model metadata:
// GET /api/model/:model/text/all
// Model digest-or-name must specified, if multiple models with same name exist only one is returned.
// Text rows returned in all languages.
func modelAllTextHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")

	// find model language-neutral metadata by digest or name
	mf := &ModelMetaFull{}

	m, ok := theCatalog.ModelMetaByDigestOrName(dn)
	if !ok {
		jsonResponse(w, r, mf)
		return // empty result: digest not found
	}
	mf.ModelMeta = *m

	// find model language-specific metadata by digest
	if t, ok := theCatalog.ModelMetaAllTextByDigest(mf.ModelMeta.Model.Digest); ok {
		mf.ModelTxtMeta = *t
	}

	// write json response
	jsonResponse(w, r, mf)
}

// langListHandler return list of model langauages:
// GET /api/model/:model/lang-list
// Model digest-or-name must specified, if multiple models with same name exist only one is returned.
func langListHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")

	m, _ := theCatalog.LangListByDigestOrName(dn)
	jsonResponse(w, r, m)
}

// wordListHandler return list of model "words": arrays of rows from lang_word and model_word db tables.
// GET /api/model/:model/word-list
// GET /api/model/:model/word-list/lang/:lang
// Model digest-or-name must specified, if multiple models with same name exist only one is returned.
// If optional lang specified then result in that language else in browser language or model default.
func wordListHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	rqLangTags := getRequestLang(r, "lang") // get optional language argument and languages accepted by browser

	wl, _ := theCatalog.WordListByDigestOrName(dn, rqLangTags)
	jsonResponse(w, r, wl)
}

// modelProfileHandler return profile db rows by model digest-or-name and profile name:
// GET /api/model/:model/profile/:profile
// If no such profile exist in database then empty result returned.
func modelProfileHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	profile := getRequestParam(r, "profile")

	p, _ := theCatalog.ModelProfileByName(dn, profile)
	jsonResponse(w, r, p)
}

// modelProfileListHandler return profile db rows by model digest-or-name:
// GET /api/model/:model/profile-list
// This is a list of profiles from model database, it is not a "model" profile(s).
// There is no explicit link between profile and model, profile can be applicable to multiple models.
func modelProfileListHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")

	pl, _ := theCatalog.ProfileNamesByDigestOrName(dn)
	jsonResponse(w, r, pl)
}

// runListHandler return list of run_lst db rows by model digest-or-name:
// GET /api/model/:model/run-list
// If multiple models with same name exist only one is returned.
func runListHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")

	rpl, _ := theCatalog.RunPubList(dn)
	jsonResponse(w, r, rpl)
}

// runListTextHandler return list of run_lst and run_txt db rows by model digest-or-name:
// GET /api/model/:model/run-list/text
// GET /api/model/:model/run-list/text/lang/:lang
// If multiple models with same name exist only one is returned.
// If optional lang specified then result in that language else in browser language.
func runListTextHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	rqLangTags := getRequestLang(r, "lang") // get optional language argument and languages accepted by browser

	rpl, _ := theCatalog.RunListText(dn, rqLangTags)
	jsonResponse(w, r, rpl)
}

// runStatusHandler return run_lst db row by model digest-or-name and run digest-or-stamp-or-name:
// GET /api/model/:model/run/:run/status
// If multiple models with same name exist then result is undefined.
// If multiple runs with same stamp or name exist then result is undefined.
// If no such run exist in database then empty result returned.
func runStatusHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	rdsn := getRequestParam(r, "run")

	rst, _ := theCatalog.RunStatus(dn, rdsn)
	jsonResponse(w, r, rst)
}

// runStatusListHandler return list run_lst db rows by model digest-or-name and run digest-or-stamp-or-name:
// GET /api/model/:model/run/:run/status/list
// If no such run exist in database then empty result returned.
func runStatusListHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	rdsn := getRequestParam(r, "run")

	rst, _ := theCatalog.RunStatusList(dn, rdsn)
	jsonResponse(w, r, rst)
}

// firstRunStatusHandler return first run_lst db row by model digest-or-name:
// GET /api/model/:model/run/status/first
// If multiple models or runs with same name exist only one is returned.
// If no run exist in database then empty result returned.
func firstRunStatusHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")

	rst, _ := theCatalog.FirstOrLastRunStatus(dn, true, false)
	jsonResponse(w, r, rst)
}

// lastRunStatusHandler return last run_lst db row by model digest-or-name:
// GET /api/model/:model/run/status/last
// If multiple models or runs with same name exist only one is returned.
// If no run exist in database then empty result returned.
func lastRunStatusHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")

	rst, _ := theCatalog.FirstOrLastRunStatus(dn, false, false)
	jsonResponse(w, r, rst)
}

// lastCompletedRunStatusHandler return last compeleted run_lst db row by model digest-or-name:
// GET /api/model/:model/run/status/last-completed
// Run completed if run status one of: s=success, x=exit, e=error
// If multiple models or runs with same name exist only one is returned.
// If no run exist in database then empty result returned.
func lastCompletedRunStatusHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")

	rst, _ := theCatalog.FirstOrLastRunStatus(dn, false, true)
	jsonResponse(w, r, rst)
}

// runFullHandler return run metadata: run_lst, run_options, run_progress, run_parameter db rows
// by model digest-or-name and digest-or-stamp-or-name:
// GET /api/model/:model/run/:run
// If multiple models with same name exist then result is undefined.
// If multiple runs with same stamp or name exist then result is undefined.
func runFullHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	rdsn := getRequestParam(r, "run")

	rp, _ := theCatalog.RunFull(dn, rdsn)
	jsonResponse(w, r, rp)
}

// runTextHandler return full run metadata: run_lst, run_options, run_progress, run_parameter db rows
// and corresponding text db rows from run_txt and run_parameter_txt tables
// by model digest-or-name and digest-or-stamp-or-name and language:
// GET /api/model/:model/run/:run/text
// GET /api/model/:model/run/:run/text/lang/:lang
// If multiple models with same name exist then result is undefined.
// If multiple runs with same stamp or name exist then result is undefined.
// It does not return non-completed runs (run in progress).
// Run completed if run status one of: s=success, x=exit, e=error.
// If optional lang specified then result in that language else in browser language.
func runTextHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	rdsn := getRequestParam(r, "run")
	rqLangTags := getRequestLang(r, "lang") // get optional language argument and languages accepted by browser

	rp, _ := theCatalog.RunTextFull(dn, rdsn, false, rqLangTags)
	jsonResponse(w, r, rp)
}

// runAllTextHandler return full run metadata: run_lst, run_options, run_progress, run_parameter db rows
// and corresponding text db rows from run_txt and run_parameter_txt tables
// by model digest-or-name and digest-or-stamp-or-name:
// GET /api/model/:model/run/:run/text/all
// If multiple models with same name exist then result is undefined.
// If multiple runs with same stamp or name exist then result is undefined.
// It does not return non-completed runs (run in progress).
// Run completed if run status one of: s=success, x=exit, e=error.
// Text rows returned in all languages.
func runAllTextHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	rdsn := getRequestParam(r, "run")

	rp, _ := theCatalog.RunTextFull(dn, rdsn, true, nil)
	jsonResponse(w, r, rp)
}

// worksetListHandler return list of workset_lst db rows by model digest-or-name:
// GET /api/model/:model/workset-list
// If multiple models with same name exist only one is returned.
func worksetListHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")

	wpl, _ := theCatalog.WorksetPubList(dn)
	jsonResponse(w, r, wpl)
}

// worksetListTextHandler return list of workset_lst and workset_txt db rows by model digest-or-name:
// GET /api/model/:model/workset-list/text
// GET /api/model/:model/workset-list/text/lang/:lang
// If multiple models with same name exist only one is returned.
// If optional lang specified then result in that language else in browser language.
func worksetListTextHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	rqLangTags := getRequestLang(r, "lang") // get optional language argument and languages accepted by browser

	wpl, _ := theCatalog.WorksetListText(dn, rqLangTags)
	jsonResponse(w, r, wpl)
}

// worksetStatusHandler return workset_lst db row by model digest-or-name and workset name:
// GET /api/model/:model/workset/:set/status
// If multiple models with same name exist only one is returned.
// If no such workset exist in database then empty result returned.
func worksetStatusHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	wsn := getRequestParam(r, "set")

	wst, ok, notFound := theCatalog.WorksetStatus(dn, wsn)
	if !ok && notFound {
		omppLog.Log("Warning workset status not found: ", dn, ": ", wsn)
	}

	jsonResponse(w, r, wst) // return non-empty workset_lst row if no errors and workset exist
}

// worksetDefaultStatusHandler return workset_lst db row of default workset by model digest-or-name:
// GET /api/model/:model/workset/status/default
// If multiple models with same name exist only one is returned.
// If no default workset exist in database then empty result returned.
func worksetDefaultStatusHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")

	wst, _ := theCatalog.WorksetDefaultStatus(dn)
	jsonResponse(w, r, wst)
}

// worksetTextHandler return full workset metadata by model digest-or-name and workset name:
// GET /api/model/:model/workset/:set/text
// GET /api/model/:model/workset/:set/text/lang/:lang
// If multiple models with same name exist only one is returned.
// If no such workset exist in database then empty result returned.
// If optional lang specified then result in that language else in browser language.
func worksetTextHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	wsn := getRequestParam(r, "set")
	rqLangTags := getRequestLang(r, "lang") // get optional language argument and languages accepted by browser

	wp, _, _ := theCatalog.WorksetTextFull(dn, wsn, false, rqLangTags)
	jsonResponse(w, r, wp)
}

// worksetAllTextHandler return full workset metadata by model digest-or-name and workset name:
// GET /api/model/:model/workset/:set/text/all
// If multiple models with same name exist only one is returned.
// If no such workset exist in database then empty result returned.
// Text rows returned in all languages.
func worksetAllTextHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	wsn := getRequestParam(r, "set")

	wp, _, _ := theCatalog.WorksetTextFull(dn, wsn, true, nil)
	jsonResponse(w, r, wp)
}

// taskListHandler return list of task_lst db rows by model digest-or-name:
// GET /api/model/:model/task-list
// If multiple models with same name exist only one is returned.
func taskListHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")

	rpl, _ := theCatalog.TaskList(dn)
	jsonResponse(w, r, rpl)
}

// taskListTextHandler return list of task_lst and task_txt db rows by model digest-or-name:
// GET /api/model/:model/task-list/text
// GET /api/model/:model/task-list/text/lang/:lang
// If multiple models with same name exist only one is returned.
// If optional lang specified then result in that language else in browser language.
func taskListTextHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	rqLangTags := getRequestLang(r, "lang") // get optional language argument and languages accepted by browser

	tpl, _ := theCatalog.TaskListText(dn, rqLangTags)
	jsonResponse(w, r, tpl)
}

// taskSetsHandler return task_lst row and task sets by model digest-or-name and task name:
// GET /api/model/:model/task/:task/sets
// If multiple models with same name exist only one is returned.
func taskSetsHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	name := getRequestParam(r, "task")

	tpl, _ := theCatalog.TaskSets(dn, name)
	jsonResponse(w, r, tpl)
}

// taskRunsHandler return task run history from task_lst, task_run_lst, task_run_set tables by model digest-or-name and task name:
// GET /api/model/:model/task/:task/runs
// If multiple models with same name exist only one is returned.
// It does not return non-completed task runs (run in progress).
// Task run history may contain model runs and input sets (worksets) which are deleted or modified by user,
// there is no guaratntee model runs still exists or worksets contain same input parameter values
// as it was at the time of task run.
func taskRunsHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	name := getRequestParam(r, "task")

	tpl, _ := theCatalog.TaskRuns(dn, name)
	jsonResponse(w, r, tpl)
}

// taskRunStatusHandler return task_run_lst db row by model digest-or-name, task name and task run stamp or run name:
// GET /api/model/:model/task/:task/run-status/run/:run
// If multiple models or runs with same name exist only one is returned.
// If no such task or run exist in database then empty result returned.
func taskRunStatusHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	tn := getRequestParam(r, "task")
	trsn := getRequestParam(r, "run")

	rst, _ := theCatalog.TaskRunStatus(dn, tn, trsn)
	jsonResponse(w, r, rst)
}

// taskRunStatusListHandler return task_run_lst db row by model digest-or-name, task name and task run stamp or run name:
// GET /api/model/:model/task/:task/run-status/list/:run
// If no such task or run exist in database then empty result returned.
func taskRunStatusListHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	tn := getRequestParam(r, "task")
	trsn := getRequestParam(r, "run")

	rst, _ := theCatalog.TaskRunStatusList(dn, tn, trsn)
	jsonResponse(w, r, rst)
}

// firstTaskRunStatusHandler return first task_run_lst db row by model digest-or-name and task name:
// GET /api/model/:model/task/:task/run-status/first
// If multiple models with same name exist only one is returned.
// If no such task or run exist in database then empty result returned.
func firstTaskRunStatusHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	tn := getRequestParam(r, "task")

	rst, _ := theCatalog.FirstOrLastTaskRunStatus(dn, tn, true, false)
	jsonResponse(w, r, rst)
}

// lastTaskRunStatusHandler return last task_run_lst db row by model digest-or-name and task name:
// GET /api/model/:model/task/:task/run-status/last
// If multiple models with same name exist only one is returned.
// If no such task or run exist in database then empty result returned.
func lastTaskRunStatusHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	tn := getRequestParam(r, "task")

	rst, _ := theCatalog.FirstOrLastTaskRunStatus(dn, tn, false, false)
	jsonResponse(w, r, rst)
}

// lastCompletedTaskRunStatusHandler return last compeleted task_run_lst db row by model digest-or-name and task name:
// GET /api/model/:model/task/:task/run-status/last-completed
// task completed if task status one of: s=success, x=exit, e=error
// If multiple models with same name exist only one is returned.
// If no such task or run exist in database then empty result returned.
func lastCompletedTaskRunStatusHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	tn := getRequestParam(r, "task")

	rst, _ := theCatalog.FirstOrLastTaskRunStatus(dn, tn, false, true)
	jsonResponse(w, r, rst)
}

// taskTextHandler return full task metadata, description, notes, run history by model digest-or-name and task name
// from db-tables: task_lst, task_txt, task_set, task_run_lst, task_run_set and also from workset_txt, run_txt.
// GET /api/model/:model/task/:task/text
// GET /api/model/:model/task/:task/text/lang/:lang
// If multiple models with same name exist only one is returned.
// It does not return non-completed task runs (run in progress).
// Run completed if run status one of: s=success, x=exit, e=error.
// It also return description and notes for all input worksets, task run(s) workset and model runs.
// If optional lang specified then result in that language else in browser language.
func taskTextHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	tn := getRequestParam(r, "task")
	rqLangTags := getRequestLang(r, "lang") // get optional language argument and languages accepted by browser

	tp, trs, _ := theCatalog.TaskTextFull(dn, tn, false, rqLangTags)

	jsonResponse(w, r,
		&struct {
			Task *db.TaskPub
			Txt  *db.TaskRunSetTxt
		}{Task: tp, Txt: trs})
}

// taskAllTextHandler return full task metadata, description, notes, run history by model digest-or-name and task name
// from db-tables: task_lst, task_txt, task_set, task_run_lst, task_run_set and also from workset_txt, run_txt.
// GET /api/model/:model/task/:task/text/all
// If multiple models with same name exist only one is returned.
// It does not return non-completed runs (run in progress).
// Run completed if run status one of: s=success, x=exit, e=error.
// It also return description and notes for all input worksets, task run(s) workset and model runs.
// Text rows returned in all languages.
func taskAllTextHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	tn := getRequestParam(r, "task")

	tp, trs, _ := theCatalog.TaskTextFull(dn, tn, true, nil)

	jsonResponse(w, r,
		&struct {
			Task *db.TaskPub
			Txt  *db.TaskRunSetTxt
		}{Task: tp, Txt: trs})
}
